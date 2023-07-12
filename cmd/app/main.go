package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	geo "myq-teslamate-geofence/internal/geo"
	util "myq-teslamate-geofence/internal/util"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	debug      bool
	configFile string
	GetDevices bool
	cars       []*util.Car // list of all cars from all garage doors
)

func init() {
	log.SetOutput(os.Stdout)
	parseArgs()
	if !GetDevices {
		util.LoadConfig(configFile)
	}
	checkEnvVars()
	for _, garageDoor := range util.Config.GarageDoors {
		for _, car := range garageDoor.Cars {
			car.GarageDoor = garageDoor
			cars = append(cars, car)
		}
	}
}

// parse args
func parseArgs() {
	// set up flags for parsing args
	flag.StringVar(&configFile, "config", "", "location of config file")
	flag.StringVar(&configFile, "c", "", "location of config file")
	flag.BoolVar(&util.Config.Testing, "testing", false, "test case")
	flag.BoolVar(&GetDevices, "d", false, "get myq devices")
	flag.Parse()

	// only check for config if not getting devices
	if !GetDevices {
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
	}
}

func main() {
	if GetDevices {
		geo.GetGarageDoorSerials(util.Config)
		return
	}
	if value, exists := os.LookupEnv("TESTING"); exists {
		util.Config.Testing, _ = strconv.ParseBool(value)
	}
	if value, exists := os.LookupEnv("DEBUG"); exists {
		debug, _ = strconv.ParseBool(value)
	}
	fmt.Println()

	// create a new MQTT client
	opts := mqtt.NewClientOptions()
	opts.SetOrderMatters(false)
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", util.Config.Global.MqttHost, util.Config.Global.MqttPort))
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetClientID(util.Config.Global.MqttClientID)

	// create a new MQTT client object
	client := mqtt.NewClient(opts)

	// connect to the MQTT broker
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("could not connect to mqtt broker: %v", token.Error())
	} else {
		log.Println("Connected to MQTT broker")
	}

	messageChan := make(chan mqtt.Message)

	// create channels to receive messages
	for _, car := range cars {
		log.Printf("Subscribing to MQTT geofence, latitude, and longitude topics for car %d", car.ID)

		for _, topic := range []string{"geofence", "latitude", "longitude"} {
			if token := client.Subscribe(
				fmt.Sprintf("teslamate/cars/%d/%s", car.ID, topic),
				0,
				func(client mqtt.Client, message mqtt.Message) {
					messageChan <- message
				}); token.Wait() && token.Error() != nil {
				log.Fatalf("%v", token.Error())
			}
		}
	}

	log.Println("Topics subscribed, listening for events...")

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
				log.Printf("Received geo for car %d: %v", car.ID, string(message.Payload()))
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
}
