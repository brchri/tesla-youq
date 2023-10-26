package ratgdo

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

type Ratgdo struct {
	Mqtt struct {
		Host          string `yaml:"host"`
		Port          int    `yaml:"port"`
		ClientID      string `yaml:"client_id"`
		User          string `yaml:"user"`
		Pass          string `yaml:"pass"`
		UseTls        bool   `yaml:"use_tls"`
		SkipTlsVerify bool   `yaml:"skip_tls_verify"`
		Prefix        string `yaml:"prefix"`
	} `yaml:"mqtt_settings"`
	MqttClient   mqtt.Client
	State        string
	Availability string
	Obstruction  string
}

var (
	topicSuffixes = []string{"status/door", "status/obstruction", "status/availability"}

	// define required start and stop states for each possible action
	states = map[string]struct {
		requiredStopState  string
		requiredStartState string
	}{
		"open": {
			requiredStopState:  "open",
			requiredStartState: "closed",
		},
		"close": {
			requiredStopState:  "closed",
			requiredStartState: "open",
		},
	}
)

const doorCommandTopic = "command/door"

func init() {
	logger.SetFormatter(&util.CustomFormatter{})
	logger.SetOutput(os.Stdout)
	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
		logger.SetLevel(logger.DebugLevel)
	}
}

func Initialize(config map[string]interface{}) (*Ratgdo, error) {
	var ratgdo *Ratgdo
	// marshall map[string]interface into yaml, then unmarshal to object based on yaml def in struct
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		logger.Fatal("Failed to marhsal garage doors yaml object")
	}
	err = yaml.Unmarshal(yamlData, &ratgdo)
	if err != nil {
		logger.Fatal("Failed to unmarhsal garage doors yaml object")
	}

	ratgdo.initializeMqttClient()

	// // set conditional MQTT client opts
	// if ratgdo.Mqtt.ClientID != "" {
	// 	logger.Debugf(" ClientID: %s", util.Config.Global.MqttClientID)
	// } else {
	// 	// generate UUID for mqtt client connection if not specified in config file
	// 	id := uuid.New().String()
	// 	logger.Debugf(" ClientID: %s", id)
	// 	opts.SetClientID(id)
	// }

	return ratgdo, nil
}

func (r *Ratgdo) initializeMqttClient() {

	logger.Debug("Setting Ratgdo MQTT Opts:")
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
	if r.Mqtt.User != "" {
		logger.Debug(" Username: true <redacted value>")
	} else {
		logger.Debug(" Username: false (not set)")
	}
	opts.SetUsername(r.Mqtt.User) // if not defined, will just set empty strings and won't be used by pkg
	if r.Mqtt.Pass != "" {
		logger.Debug(" Password: true <redacted value>")
	} else {
		logger.Debug(" Password: false (not set)")
	}
	opts.SetPassword(r.Mqtt.Pass) // if not defined, will just set empty strings and won't be used by pkg
	opts.OnConnect = r.onMqttConnect

	// set conditional MQTT client opts
	if r.Mqtt.ClientID != "" {
		logger.Debugf(" ClientID: %s", r.Mqtt.ClientID)
		opts.SetClientID(r.Mqtt.ClientID)
	} else {
		// generate UUID for mqtt client connection if not specified in config file
		id := uuid.New().String()
		logger.Debugf(" ClientID: %s", id)
		opts.SetClientID(id)
	}
	logger.Debug(" Protocol: TCP")
	mqttProtocol := "tcp"
	if r.Mqtt.UseTls {
		logger.Debug(" UseTLS: true")
		logger.Debugf(" SkipTLSVerify: %t", r.Mqtt.SkipTlsVerify)
		opts.SetTLSConfig(&tls.Config{
			InsecureSkipVerify: r.Mqtt.SkipTlsVerify,
		})
		mqttProtocol = "ssl"
	} else {
		logger.Debug(" UseTLS: false")
	}
	broker := fmt.Sprintf("%s://%s:%d", mqttProtocol, r.Mqtt.Host, r.Mqtt.Port)
	logger.Debugf(" Broker: %s", broker)
	opts.AddBroker(broker)

	// create a new MQTT client object
	r.MqttClient = mqtt.NewClient(opts)

	// connect to the MQTT broker
	logger.Debug("Connecting to Ratgdo MQTT broker")
	if token := r.MqttClient.Connect(); token.Wait() && token.Error() != nil {
		logger.Fatalf("could not connect to mqtt broker: %v", token.Error())
	} else {
		logger.Info("Connected to MQTT broker")
	}
	logger.Debugf("MQTT Broker Connected: %t", r.MqttClient.IsConnected())
}

func (r *Ratgdo) onMqttConnect(client mqtt.Client) {
	for _, t := range topicSuffixes {
		fullTopic := r.Mqtt.Prefix + "/" + t
		logger.Debugf("Subscribing to Ratgdo MQTT topic %s", fullTopic)
		topicSubscribed := false
		// retry topic subscription attempts with 1 sec delay between attempts
		for retryAttempts := 5; retryAttempts > 0; retryAttempts-- {
			logger.Debugf("Subscribing to topic: %s", fullTopic)
			if token := client.Subscribe(
				fullTopic,
				0,
				r.processMqttMessage); token.Wait() && token.Error() == nil {
				topicSubscribed = true
				logger.Debugf("Topic subscribed successfully: %s", fullTopic)
				break
			} else {
				logger.Infof("Failed to subscribe to topic %s for car ratgdo, will make %d more attempts. Error: %v", fullTopic, retryAttempts, token.Error())
			}
			time.Sleep(5 * time.Second)
		}
		if !topicSubscribed {
			logger.Fatalf("Unable to subscribe to topics, exiting")
		}
	}
	logger.Debug("Ratgdo topics subscribed, listening for events...")
}

func (r *Ratgdo) processMqttMessage(client mqtt.Client, message mqtt.Message) {
	m := strings.Split(message.Topic(), "/")
	switch m[len(m)-1] {
	case "door":
		r.State = string(message.Payload())
	case "availability":
		r.Availability = string(message.Payload())
	case "obstruction":
		r.Obstruction = string(message.Payload())
	default:
		logger.Debugf("invalid message topic: %s", message.Topic())
	}
}

func (r *Ratgdo) SetGarageDoor(action string) (err error) {

	if r.State != states[action].requiredStartState {
		logger.Warnf("Action and state mismatch: garage state is not valid for executing requested action; current state: %s; requested action: %s", r.State, action)
		return
	}

	logger.Infof("setting garage door %s", action)
	logger.Debugf("Reported Ratgdo availability: %s", r.Availability)

	token := r.MqttClient.Publish(r.Mqtt.Prefix+"/"+doorCommandTopic, 0, false, action)
	token.Wait()

	time.Sleep(3 * time.Second)

	// wait 30 seconds to reach desired state
	start := time.Now()
	for time.Since(start) < 30*time.Second {
		if r.State == states[action].requiredStopState {
			logger.Infof("Garage door state has been set successfully: %s", action)
			return
		}
		logger.Debugf("Current opener state: %s", r.State)
		time.Sleep(1 * time.Second)
	}

	if r.Availability == "offline" {
		err = fmt.Errorf("unable to %s garage door, possible reason: ratgdo availability reporting offline", action)
	} else if r.Obstruction == "obstructed" {
		err = fmt.Errorf("unable to %s garage door, possible reason: ratgdo obstruction reported", action)
	} else {
		err = fmt.Errorf("unable to %s garage door, possible reason: unknown; current state: %s", action, r.State)
	}

	return
}
