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

	util "github.com/brchri/tesla-youq/internal/util"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	debug       bool
	configFile  string
	cars        []*util.Car                  // list of all cars from all garage doors
	version     string            = "v0.0.1" // pass -ldflags="-X main.version=<version>" at build time to set linker flag and bake in binary version
	messageChan chan mqtt.Message            // channel to receive mqtt messages
)

func init() {
	log.SetOutput(os.Stdout)
	parseArgs()
	util.LoadConfig(configFile)
	checkEnvVars()
	for _, garageDoor := range util.Config.GarageDoors {
		for _, car := range garageDoor.Cars {
			car.GarageDoor = garageDoor
			cars = append(cars, car)
			car.IsInsidePolygonGeo = true // default to within geofence for polygon, even if polygon not used
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
				log.Fatalf("Config file must be defined with '-c' or 'CONFIG_FILE' environment variable")
			}
		}

		// check that ConfigFile exists
		if _, err := os.Stat(configFile); err != nil {
			log.Fatalf("Config file %v doesn't exist!", configFile)
		}
	} else {
		// if -d flag passed, get devices and exit
		geo.GetGarageDoorSerials(util.Config)
		os.Exit(0)
	}
}

func main() {
	if value, exists := os.LookupEnv("TESTING"); exists {
		util.Config.Testing, _ = strconv.ParseBool(value)
	}
	if value, exists := os.LookupEnv("DEBUG"); exists {
		debug, _ = strconv.ParseBool(value)
	}
	fmt.Println()

	messageChan = make(chan mqtt.Message)

	// create a new MQTT client
	opts := mqtt.NewClientOptions()
	opts.SetOrderMatters(false)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetUsername(util.Config.Global.MqttUser) // if not defined, will just set empty strings and won't be used by pkg
	opts.SetPassword(util.Config.Global.MqttPass) // if not defined, will just set empty strings and won't be used by pkg
	opts.OnConnect = onMqttConnect

	// set conditional MQTT client opts
	if util.Config.Global.MqttClientID != "" {
		opts.SetClientID(util.Config.Global.MqttClientID)
	} else {
		// generate UUID for mqtt client connection if not specified in config file
		opts.SetClientID(uuid.New().String())
	}
	mqttProtocol := "tcp"
	if util.Config.Global.MqttUseTls {
		opts.SetTLSConfig(&tls.Config{
			InsecureSkipVerify: util.Config.Global.MqttSkipTlsVerify,
		})
		mqttProtocol = "ssl"
	}
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", mqttProtocol, util.Config.Global.MqttHost, util.Config.Global.MqttPort))

	// create a new MQTT client object
	client := mqtt.NewClient(opts)

	// connect to the MQTT broker
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("could not connect to mqtt broker: %v", token.Error())
	} else {
		log.Println("Connected to MQTT broker")
	}

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
				log.Printf("Received geo for car %d: %v", car.ID, car.CurGeofence)
				go geo.CheckGeoFence(util.Config, car)
			case "latitude":
				if debug {
					log.Printf("Received lat for car %d: %v", car.ID, string(message.Payload()))
				}
				car.CurLat, _ = strconv.ParseFloat(string(message.Payload()), 64)
				go geo.CheckGeoFence(util.Config, car)
			case "longitude":
				if debug {
					log.Printf("Received long for car %d: %v", car.ID, string(message.Payload()))
				}
				car.CurLng, _ = strconv.ParseFloat(string(message.Payload()), 64)
				go geo.CheckGeoFence(util.Config, car)
			}

		case <-signalChannel:
			log.Println("Received interrupt signal, shutting down...")
			client.Disconnect(250)
			time.Sleep(250 * time.Millisecond)
			return

		}
	}
}

// subscribe to topics when MQTT client connects (or reconnects)
func onMqttConnect(client mqtt.Client) {
	for _, car := range cars {
		log.Printf("Subscribing to MQTT topics for car %d", car.ID)

		// define which topics are relevant for each car based on config
		var topics []string
		if car.GarageDoor.IsPolygonGeofenceDefined() {
			car.GarageDoor.GeofenceType = util.PolygonGeofence
			topics = []string{"latitude", "longitude"}
		} else if car.GarageDoor.TriggerCloseGeofence.IsGeofenceDefined() && car.GarageDoor.TriggerOpenGeofence.IsGeofenceDefined() {
			car.GarageDoor.GeofenceType = util.TeslamateGeofence
			topics = []string{"geofence"}
		} else if car.GarageDoor.Location.IsPointDefined() {
			topics = []string{"latitude", "longitude"}
			car.GarageDoor.GeofenceType = util.DistanceGeofence
		} else {
			log.Fatalf("must define a valid location and radii for garage door or open and close geofence triggers")
		}

		// subscribe to topics
		for _, topic := range topics {
			topicSubscribed := false
			// retry topic subscription attempts with 1 sec delay between attempts
			for retryAttempts := 5; retryAttempts > 0; retryAttempts-- {
				if token := client.Subscribe(
					fmt.Sprintf("teslamate/cars/%d/%s", car.ID, topic),
					0,
					func(client mqtt.Client, message mqtt.Message) {
						messageChan <- message
					}); token.Wait() && token.Error() == nil {
					topicSubscribed = true
					break
				} else {
					log.Printf("Failed to subscribe to topic %s for car %d, will make %d more attempts. Error: %v", topic, car.ID, retryAttempts, token.Error())
				}
				time.Sleep(5 * time.Second)
			}
			if !topicSubscribed {
				log.Fatalf("Unable to subscribe to topics, exiting")
			}
		}
	}

	log.Println("Topics subscribed, listening for events...")
}

// check for env vars and validate that a myq_email and myq_pass exists
func checkEnvVars() {
	// override config with env vars if present
	if value, exists := os.LookupEnv("MYQ_EMAIL"); exists {
		util.Config.Global.MyQEmail = value
	}
	if value, exists := os.LookupEnv("MYQ_PASS"); exists {
		util.Config.Global.MyQPass = value
	}
	if util.Config.Global.MyQEmail == "" || util.Config.Global.MyQPass == "" {
		log.Fatal("MYQ_EMAIL and MYQ_PASS must be defined in the config file or as env vars")
	}
	if value, exists := os.LookupEnv("MQTT_USER"); exists {
		util.Config.Global.MqttUser = value
	}
	if value, exists := os.LookupEnv("MQTT_PASS"); exists {
		util.Config.Global.MqttPass = value
	}
}
