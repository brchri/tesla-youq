package mqtt

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/brchri/tesla-youq/internal/util"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type (
	MqttGdo struct {
		MqttSettings struct {
			Connection struct {
				Host          string `yaml:"host"`
				Port          int    `yaml:"port"`
				ClientID      string `yaml:"client_id"`
				User          string `yaml:"user"`
				Pass          string `yaml:"pass"`
				UseTls        bool   `yaml:"use_tls"`
				SkipTlsVerify bool   `yaml:"skip_tls_verify"`
			} `yaml:"connection"`
			Topics struct {
				Prefix       string `yaml:"prefix"`
				DoorStatus   string `yaml:"door_status"`
				Obstruction  string `yaml:"obstruction"`
				Availability string `yaml:"availability"`
			} `yaml:"topics"`
			Commands []Command `yaml:"commands"`
		} `yaml:"mqtt_settings"`
		ModuleName   string `yaml:"module_name"`
		MqttClient   mqtt.Client
		State        string
		Availability string
		Obstruction  string
	}

	Command struct {
		Name               string `yaml:"name"`    // e.g. `open` or `close`
		Payload            string `yaml:"payload"` // this could be the same or different than Name depending on the mqtt implementation
		TopicSuffix        string `yaml:"topic_suffix"`
		RequiredStartState string `yaml:"required_start_state"`
		RequiredStopState  string `yaml:"required_stop_state"`
	}
)

const defaultModuleName = "Generic MQTT Opener"

func init() {
	logger.SetFormatter(&util.CustomFormatter{})
	logger.SetOutput(os.Stdout)
	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
		logger.SetLevel(logger.DebugLevel)
	}
}

func Initialize(config map[string]interface{}) (*MqttGdo, error) {
	var mqttGdo *MqttGdo
	// marshall map[string]interface into yaml, then unmarshal to object based on yaml def in struct
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		logger.Fatal("Failed to marhsal garage doors yaml object")
	}
	err = yaml.Unmarshal(yamlData, &mqttGdo)
	if err != nil {
		logger.Fatal("Failed to unmarhsal garage doors yaml object")
	}

	if mqttGdo.ModuleName == "" {
		mqttGdo.ModuleName = defaultModuleName
	}

	// check if garage door opener is connecting to the same mqtt broker as the global for teslamate, and if so, that they have unique clientIDs
	localMqtt := &mqttGdo.MqttSettings.Connection
	globalMqtt := util.Config.Global.MqttSettings.Connection
	if localMqtt.ClientID != "" && localMqtt.ClientID == globalMqtt.ClientID && localMqtt.Host == globalMqtt.Host && localMqtt.Port == globalMqtt.Port {
		localMqtt.ClientID = localMqtt.ClientID + "-" + uuid.NewString()
		logger.Warnf("mqtt client id for door opener is the same as the global, appending random uuid to the name: %s", localMqtt.ClientID)
	}

	mqttGdo.MqttSettings.Topics.Prefix = strings.TrimRight(mqttGdo.MqttSettings.Topics.Prefix, "/") // trim any trailing `/` on the prefix topic

	mqttGdo.initializeMqttClient()

	return mqttGdo, nil
}

func (m *MqttGdo) initializeMqttClient() {

	logger.Debug("Setting MqttGdo MQTT Opts:")
	// create a new MQTT client
	opts := mqtt.NewClientOptions()
	logger.Debug(" OrderMatters: false")
	opts.SetOrderMatters(false)
	logger.Debug(" KeepAlive: 30 seconds")
	opts.SetKeepAlive(30 * time.Second)
	logger.Debug(" PingTimeout: 10 seconds")
	opts.SetPingTimeout(10 * time.Second)
	logger.Debug(" AutoReconnect: true")
	opts.SetAutoReconnect(true)
	if m.MqttSettings.Connection.User != "" {
		logger.Debug(" Username: true <redacted value>")
	} else {
		logger.Debug(" Username: false (not set)")
	}
	opts.SetUsername(m.MqttSettings.Connection.User) // if not defined, will just set empty strings and won't be used by pkg
	if m.MqttSettings.Connection.Pass != "" {
		logger.Debug(" Password: true <redacted value>")
	} else {
		logger.Debug(" Password: false (not set)")
	}
	opts.SetPassword(m.MqttSettings.Connection.Pass) // if not defined, will just set empty strings and won't be used by pkg
	opts.OnConnect = m.onMqttConnect

	// set conditional MQTT client opts
	if m.MqttSettings.Connection.ClientID != "" {
		logger.Debugf(" ClientID: %s", m.MqttSettings.Connection.ClientID)
		opts.SetClientID(m.MqttSettings.Connection.ClientID)
	} else {
		// generate UUID for mqtt client connection if not specified in config file
		id := uuid.New().String()
		logger.Debugf(" ClientID: %s", id)
		opts.SetClientID(id)
	}
	logger.Debug(" Protocol: TCP")
	mqttProtocol := "tcp"
	if m.MqttSettings.Connection.UseTls {
		logger.Debug(" UseTLS: true")
		logger.Debugf(" SkipTLSVerify: %t", m.MqttSettings.Connection.SkipTlsVerify)
		opts.SetTLSConfig(&tls.Config{
			InsecureSkipVerify: m.MqttSettings.Connection.SkipTlsVerify,
		})
		mqttProtocol = "ssl"
	} else {
		logger.Debug(" UseTLS: false")
	}
	broker := fmt.Sprintf("%s://%s:%d", mqttProtocol, m.MqttSettings.Connection.Host, m.MqttSettings.Connection.Port)
	logger.Debugf(" Broker: %s", broker)
	opts.AddBroker(broker)

	// create a new MQTT client object
	m.MqttClient = mqtt.NewClient(opts)

	// connect to the MQTT broker
	logger.Debug("Connecting to MqttGdo MQTT broker")
	if token := m.MqttClient.Connect(); token.Wait() && token.Error() != nil {
		logger.Fatalf("could not connect to mqtt broker: %v", token.Error())
	} else {
		logger.Infof("%s door opener connected to MQTT broker", m.ModuleName)
	}
	logger.Debugf("MQTT Broker Connected: %t", m.MqttClient.IsConnected())
}

func (m *MqttGdo) onMqttConnect(client mqtt.Client) {
	topicSuffixes := []string{
		m.MqttSettings.Topics.Obstruction,
		m.MqttSettings.Topics.Availability,
		m.MqttSettings.Topics.DoorStatus,
	}

	for _, t := range topicSuffixes {
		if t == "" {
			// don't process if any of the suffixes are empty
			continue
		}

		fullTopic := m.MqttSettings.Topics.Prefix + "/" + t
		logger.Debugf("Subscribing to MqttGdo MQTT topic %s", fullTopic)
		topicSubscribed := false
		// retry topic subscription attempts with 1 sec delay between attempts
		for retryAttempts := 5; retryAttempts > 0; retryAttempts-- {
			logger.Debugf("Subscribing to topic: %s", fullTopic)
			if token := client.Subscribe(
				fullTopic,
				0,
				m.processMqttMessage); token.Wait() && token.Error() == nil {
				topicSubscribed = true
				logger.Debugf("Topic subscribed successfully: %s", fullTopic)
				break
			} else {
				logger.Infof("Failed to subscribe to topic %s for car mqttGdo, will make %d more attempts. Error: %v", fullTopic, retryAttempts, token.Error())
			}
			time.Sleep(5 * time.Second)
		}
		if !topicSubscribed {
			logger.Fatalf("Unable to subscribe to topics, exiting")
		}
	}
	logger.Debug("MqttGdo topics subscribed, listening for events...")
}

func (m *MqttGdo) processMqttMessage(client mqtt.Client, message mqtt.Message) {
	// update MqttGdo property based on topic suffix (strip shared prefix on the switch)
	switch strings.TrimPrefix(message.Topic(), m.MqttSettings.Topics.Prefix+"/") {
	case m.MqttSettings.Topics.DoorStatus:
		m.State = string(message.Payload())
	case m.MqttSettings.Topics.Availability:
		m.Availability = string(message.Payload())
	case m.MqttSettings.Topics.Obstruction:
		m.Obstruction = string(message.Payload())
	default:
		logger.Debugf("invalid message topic: %s", message.Topic())
	}
}

func (m *MqttGdo) SetGarageDoor(action string) (err error) {
	var command Command
	for _, v := range m.MqttSettings.Commands {
		if action == v.Name {
			command = v
			break
		}
	}

	if command.Name == "" {
		return fmt.Errorf("no command defined for action %s", action)
	}

	// if status topic and required state are defined, check that the required state is satisfied
	if m.MqttSettings.Topics.DoorStatus != "" && command.RequiredStartState != "" && m.State != command.RequiredStartState {
		logger.Warnf("Action and state mismatch: garage state is not valid for executing requested action; current state: %s; requested action: %s", m.State, action)
		return
	}

	if util.Config.Testing {
		logger.Infof("TESTING flag set - Would attempt action %v", action)
		return
	}

	logger.Infof("setting garage door %s", action)
	logger.Debugf("Reported MqttGdo availability: %s", m.Availability)

	token := m.MqttClient.Publish(m.MqttSettings.Topics.Prefix+"/"+command.TopicSuffix, 0, false, command.Payload)
	token.Wait()

	time.Sleep(3 * time.Second)

	// if a required stop state and status topic are defined, wait for it to be satisfied
	if command.RequiredStopState != "" && m.MqttSettings.Topics.DoorStatus != "" {
		// wait 30 seconds to reach desired state
		start := time.Now()
		for time.Since(start) < 30*time.Second {
			if m.State == command.RequiredStopState {
				logger.Infof("Garage door state has been set successfully: %s", action)
				return
			}
			logger.Debugf("Current opener state: %s", m.State)
			time.Sleep(1 * time.Second)
		}

		// these are based on ratgdo implementation, should probably make them configurable as other implementations may not use the same statuses
		if m.MqttSettings.Topics.Availability != "" && m.Availability == "offline" {
			err = fmt.Errorf("unable to %s garage door, possible reason: mqttGdo availability reporting offline", action)
		} else if m.MqttSettings.Topics.Obstruction != "" && m.Obstruction == "obstructed" {
			err = fmt.Errorf("unable to %s garage door, possible reason: mqttGdo obstruction reported", action)
		} else {
			err = fmt.Errorf("unable to %s garage door, possible reason: unknown; current state: %s", action, m.State)
		}
	} else {
		logger.Infof("Garage door command `%s` has been published to the topic", action)
	}

	return
}
