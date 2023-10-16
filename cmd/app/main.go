package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	geo "github.com/brchri/tesla-youq/internal/geo"
	"github.com/google/uuid"
	logger "github.com/sirupsen/logrus"

	util "github.com/brchri/tesla-youq/internal/util"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	configFile  string
	cars        []*util.Car                  // list of all cars from all garage doors
	version     string            = "v0.0.1" // pass -ldflags="-X main.version=<version>" at build time to set linker flag and bake in binary version
	messageChan chan mqtt.Message            // channel to receive mqtt messages
)

func init() {
	logger.SetFormatter(&util.CustomFormatter{})
	if val, ok := os.LookupEnv("DEBUG"); ok && strings.ToLower(val) == "true" {
		logger.SetLevel(logger.DebugLevel)
	}
	log.SetOutput(os.Stdout)
	parseArgs()
	util.LoadConfig(configFile)
	checkEnvVars()
	for _, garageDoor := range util.Config.GarageDoors {
		for _, car := range garageDoor.Cars {
			car.GarageDoor = garageDoor
			cars = append(cars, car)
			if car.GarageDoor.GeofenceType == util.PolygonGeofenceType {
				car.InsidePolyCloseGeo = true
				car.InsidePolyOpenGeo = true
			}
			// start listening to car update location channels
			go processLocationUpdates(car)
		}
	}
}

// parse args
func parseArgs() {
	// set up flags for parsing args
	var getDevices bool
	var getVersion bool
	flag.StringVar(&configFile, "config", "", "location of config file")
	flag.StringVar(&configFile, "c", "", "location of config file")
	flag.BoolVar(&util.Config.Testing, "testing", false, "test case")
	flag.BoolVar(&getDevices, "d", false, "get myq devices")
	flag.BoolVar(&getVersion, "v", false, "print version info and return")
	flag.BoolVar(&getVersion, "version", false, "print version info and return")
	flag.Parse()

	if getVersion {
		versionInfo := filepath.Base(os.Args[0]) + " " + version + " " + runtime.GOOS + "/" + runtime.GOARCH
		fmt.Println(versionInfo)
		os.Exit(0)
	}

	// only check for config if not getting devices
	if !getDevices {
		// if -c or --config wasn't passed, check for CONFIG_FILE env var
		// if that fails, check for file at default location
		if configFile == "" {
			var exists bool
			if configFile, exists = os.LookupEnv("CONFIG_FILE"); !exists {
				logger.Fatalf("Config file must be defined with '-c' or 'CONFIG_FILE' environment variable")
			}
		}

		// check that ConfigFile exists
		if _, err := os.Stat(configFile); err != nil {
			logger.Fatalf("Config file %v doesn't exist!", configFile)
		}
	} else {
		// if -d flag passed, get devices and exit
		checkEnvVars()
		geo.GetGarageDoorSerials(util.Config)
		os.Exit(0)
	}
}

func main() {
	messageChan = make(chan mqtt.Message)

	logger.Debug("Setting MQTT Opts:")
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
	if util.Config.Global.MqttUser != "" {
		logger.Debug(" Username: true <redacted value>")
	} else {
		logger.Debug(" Username: false (not set)")
	}
	opts.SetUsername(util.Config.Global.MqttUser) // if not defined, will just set empty strings and won't be used by pkg
	if util.Config.Global.MqttPass != "" {
		logger.Debug(" Password: true <redacted value>")
	} else {
		logger.Debug(" Password: false (not set)")
	}
	opts.SetPassword(util.Config.Global.MqttPass) // if not defined, will just set empty strings and won't be used by pkg
	opts.OnConnect = onMqttConnect

	// set conditional MQTT client opts
	if util.Config.Global.MqttClientID != "" {
		logger.Debugf(" ClientID: %s", util.Config.Global.MqttClientID)
		opts.SetClientID(util.Config.Global.MqttClientID)
	} else {
		// generate UUID for mqtt client connection if not specified in config file
		id := uuid.New().String()
		logger.Debugf(" ClientID: %s", id)
		opts.SetClientID(id)
	}
	logger.Debug(" Protocol: TCP")
	mqttProtocol := "tcp"
	if util.Config.Global.MqttUseTls {
		logger.Debug(" UseTLS: true")
		logger.Debugf(" SkipTLSVerify: %t", util.Config.Global.MqttSkipTlsVerify)
		opts.SetTLSConfig(&tls.Config{
			InsecureSkipVerify: util.Config.Global.MqttSkipTlsVerify,
		})
		mqttProtocol = "ssl"
	} else {
		logger.Debug(" UseTLS: false")
	}
	broker := fmt.Sprintf("%s://%s:%d", mqttProtocol, util.Config.Global.MqttHost, util.Config.Global.MqttPort)
	logger.Debugf(" Broker: %s", broker)
	opts.AddBroker(broker)

	// create a new MQTT client object
	client := mqtt.NewClient(opts)

	// connect to the MQTT broker
	logger.Debug("Connecting to MQTT broker")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		logger.Fatalf("could not connect to mqtt broker: %v", token.Error())
	} else {
		logger.Info("Connected to MQTT broker")
	}
	logger.Debugf("MQTT Broker Connected: %t", client.IsConnected())

	// listen for incoming messages
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case message := <-messageChan:
			m := strings.Split(message.Topic(), "/")

			// locate car and car's garage door
			var car *util.Car
			for _, c := range cars {
				if fmt.Sprintf("%d", c.ID) == m[2] {
					car = c
					break
				}
			}

			// if lat or lng received, check geofence
			switch m[3] {
			case "geofence":
				car.PrevGeofence = car.CurGeofence
				car.CurGeofence = string(message.Payload())
				logger.Infof("Received geo for car %d: %v", car.ID, car.CurGeofence)
				go geo.CheckGeofence(util.Config, car)
			case "latitude":
				logger.Debugf("Received lat for car %d: %v", car.ID, string(message.Payload()))
				lat, _ := strconv.ParseFloat(string(message.Payload()), 64)
				go func(lat float64) {
					// send as goroutine so it doesn't block other vehicle updates if channel buffer is full
					car.LocationUpdate <- util.Point{Lat: lat, Lng: 0}
				}(lat)
			case "longitude":
				logger.Debugf("Received long for car %d: %v", car.ID, string(message.Payload()))
				lng, _ := strconv.ParseFloat(string(message.Payload()), 64)
				go func(lng float64) {
					// send as goroutine so it doesn't block other vehicle updates if channel buffer is full
					car.LocationUpdate <- util.Point{Lat: 0, Lng: lng}
				}(lng)
			}

		case <-signalChannel:
			logger.Info("Received interrupt signal, shutting down...")
			client.Disconnect(250)
			time.Sleep(250 * time.Millisecond)
			return

		}
	}
}

// watches the LocationUpdate channel for a car and queues a CheckGeofence operation
// this allows threaded geofence checks for multiple vehicles, while each individual vehicle
// does not have parallel threads executing checks
func processLocationUpdates(car *util.Car) {
	for update := range car.LocationUpdate {
		if update.Lat != 0 {
			car.CurrentLocation.Lat = update.Lat
		}
		if update.Lng != 0 {
			car.CurrentLocation.Lng = update.Lng
		}
		if car.CurrentLocation.IsPointDefined() {
			geo.CheckGeofence(util.Config, car)
		}
	}
}

// subscribe to topics when MQTT client connects (or reconnects)
func onMqttConnect(client mqtt.Client) {
	for _, car := range cars {
		logger.Infof("Subscribing to MQTT topics for car %d", car.ID)

		// define which topics are relevant for each car based on config
		var topics []string
		switch car.GarageDoor.GeofenceType {
		case util.PolygonGeofenceType:
			topics = []string{"latitude", "longitude"}
		case util.CircularGeofenceType:
			topics = []string{"latitude", "longitude"}
		case util.TeslamateGeofenceType:
			topics = []string{"geofence"}
		}

		// subscribe to topics
		for _, topic := range topics {
			topicSubscribed := false
			// retry topic subscription attempts with 1 sec delay between attempts
			for retryAttempts := 5; retryAttempts > 0; retryAttempts-- {
				fullTopic := fmt.Sprintf("teslamate/cars/%d/%s", car.ID, topic)
				logger.Debugf("Subscribing to topic: %s", fullTopic)
				if token := client.Subscribe(
					fullTopic,
					0,
					func(client mqtt.Client, message mqtt.Message) {
						messageChan <- message
					}); token.Wait() && token.Error() == nil {
					topicSubscribed = true
					logger.Debugf("Topic subscribed successfully: %s", fullTopic)
					break
				} else {
					logger.Infof("Failed to subscribe to topic %s for car %d, will make %d more attempts. Error: %v", topic, car.ID, retryAttempts, token.Error())
				}
				time.Sleep(5 * time.Second)
			}
			if !topicSubscribed {
				logger.Fatalf("Unable to subscribe to topics, exiting")
			}
		}
	}

	logger.Info("Topics subscribed, listening for events...")
}

// check for env vars and validate that a myq_email and myq_pass exists
func checkEnvVars() {
	logger.Debug("Checking environment variables:")
	// override config with env vars if present
	if value, exists := os.LookupEnv("MYQ_EMAIL"); exists {
		logger.Debug("  MYQ_EMAIL defined, overriding config")
		util.Config.Global.MyQEmail = value
	}
	if value, exists := os.LookupEnv("MYQ_PASS"); exists {
		logger.Debug("  MYQ_PASS defined, overriding config")
		util.Config.Global.MyQPass = value
	}
	if util.Config.Global.MyQEmail == "" || util.Config.Global.MyQPass == "" {
		logger.Fatal("  MYQ_EMAIL and MYQ_PASS must be defined in the config file or as env vars")
	}
	if value, exists := os.LookupEnv("MQTT_USER"); exists {
		logger.Debug("  MQTT_USER defined, overriding config")
		util.Config.Global.MqttUser = value
	}
	if value, exists := os.LookupEnv("MQTT_PASS"); exists {
		logger.Debug("  MQTT_PASS defined, overriding config")
		util.Config.Global.MqttPass = value
	}
	if value, exists := os.LookupEnv("TESTING"); exists {
		util.Config.Testing, _ = strconv.ParseBool(value)
		logger.Debugf("  TESTING=%t", util.Config.Testing)
	}
	if value, exists := os.LookupEnv("DEBUG"); exists {
		logger.Debugf("  DEBUG=%s", value)
	}
}
