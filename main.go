package main

import (
	"flag"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joeshaw/myq"
	"log"
	"math"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	carAtHome  = true // this is an assumption, may not be accurate on first run
	testing    bool
	debug      bool
	configFile string
	Config     ConfigStruct
)

type (
	Point struct {
		Lat float64 `yaml:"lat"`
		Lng float64 `yaml:"lng"`
	}

	Geofence struct {
		Center Point   `yaml:"geo_center"`
		Radius float64 `yaml:"geo_radius"`
	}

	Car struct {
		CarID          int      `yaml:"teslamate_car_id"`
		MyQSerial      string   `yaml:"myq_serial"`
		GarageCloseGeo Geofence `yaml:"garage_close_geofence"`
		GarageOpenGeo  Geofence `yaml:"garage_open_geofence"`
		GeoChan        chan mqtt.Message
		LatChan        chan mqtt.Message
		LngChan        chan mqtt.Message
		MqttChan       chan mqtt.Message
		CurLat         float64
		CurLng         float64
		OpLock         bool
	}

	ConfigStruct struct {
		Global struct {
			MqttHost     string `yaml:"mqtt_host"`
			MqttPort     int    `yaml:"mqtt_port"`
			MqttClientID string `yaml:"mqtt_client_id"`
			OpCooldown   int    `yaml:"cooldown"`
			MyQEmail     string `yaml:"myq_email"`
			MyQPass      string `yaml:"myq_pass"`
		} `yaml:"global"`
		Cars []*Car `yaml:"cars"`
	}
)

func init() {
	parseArgs()
	loadConfig()
}

// parse args
func parseArgs() {
	// set up flags for parsing args
	flag.StringVar(&configFile, "config", "", "location of config file")
	flag.StringVar(&configFile, "c", "", "location of config file")
	flag.BoolVar(&testing, "testing", false, "test case")
	flag.Parse()

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

// load yaml config
func loadConfig() {
	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Could not read config file: %v", err)
	}

	err = yaml.Unmarshal(yamlFile, &Config)
	if err != nil {
		log.Fatalf("Could not load yaml from config file, received error: %v", err)
	}
}

func main() {
	if value, exists := os.LookupEnv("TESTING"); exists {
		testing, _ = strconv.ParseBool(value)
	}
	if value, exists := os.LookupEnv("DEBUG"); exists {
		debug, _ = strconv.ParseBool(value)
	}
	fmt.Println()

	checkEnvVars()

	// create a new MQTT client
	opts := mqtt.NewClientOptions()
	opts.SetOrderMatters(false)
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", Config.Global.MqttHost, Config.Global.MqttPort))
	opts.SetClientID(Config.Global.MqttClientID)

	// create a new MQTT client object
	client := mqtt.NewClient(opts)

	// connect to the MQTT broker
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("could not connect to mqtt broker: %v", token.Error())
	}

	// create a channel to receive messages
	for _, car := range Config.Cars {

		car.GeoChan = make(chan mqtt.Message)
		car.LatChan = make(chan mqtt.Message)
		car.LngChan = make(chan mqtt.Message)
		car.MqttChan = make(chan mqtt.Message)

		if token := client.Subscribe(
			fmt.Sprintf("teslamate/cars/%d/geofence", car.CarID),
			0,
			func(client mqtt.Client, message mqtt.Message) {
				car.GeoChan <- message
			}); token.Wait() && token.Error() != nil {
			log.Fatalf("%v", token.Error())
		}

		if token := client.Subscribe(
			fmt.Sprintf("teslamate/cars/%d/longitude", car.CarID),
			0,
			func(client mqtt.Client, message mqtt.Message) {
				car.LngChan <- message
			}); token.Wait() && token.Error() != nil {
			log.Fatalf("%v", token.Error())
		}

		if token := client.Subscribe(
			fmt.Sprintf("teslamate/cars/%d/latitude", car.CarID),
			0,
			func(client mqtt.Client, message mqtt.Message) {
				car.LatChan <- message
			}); token.Wait() && token.Error() != nil {
			log.Fatalf("%v", token.Error())
		}
	}

	// listen for incoming messages
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	for {
		for _, car := range Config.Cars {
			select {
			case message := <-car.GeoChan:
				log.Printf("Received geo for car %d: %v", car.CarID, string(message.Payload()))

			case message := <-car.LatChan:
				if debug {
					log.Printf("Received lat for car %d: %v", car.CarID, string(message.Payload()))
				}
				car.CurLat, _ = strconv.ParseFloat(string(message.Payload()), 64)
				go checkGeoFence(car)

			case message := <-car.LngChan:
				if debug {
					log.Printf("Received long for car %d: %v", car.CarID, string(message.Payload()))
				}
				car.CurLng, _ = strconv.ParseFloat(string(message.Payload()), 64)
				go checkGeoFence(car)

			case <-signalChannel:
				log.Println("Received interrupt signal, shutting down...")
				client.Disconnect(250)
				time.Sleep(250 * time.Millisecond)
				return
			}
		}
	}
}

func setGarageDoor(deviceSerial string, action string) error {
	s := &myq.Session{}
	s.Username = Config.Global.MyQEmail
	s.Password = Config.Global.MyQPass

	var desiredState string
	switch action {
	case myq.ActionOpen:
		desiredState = myq.StateOpen
	case myq.ActionClose:
		desiredState = myq.StateClosed
	}

	if testing {
		log.Printf("Would attempt action %v", action)
		time.Sleep(3 * time.Second)
		return nil
	}

	log.Println("Attempting to get valid MyQ session...")
	if err := s.Login(); err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("ERROR: %v\n", err)
		log.SetOutput(os.Stdout)
		return err
	}
	log.Println("Session acquired...")

	curState, err := s.DeviceState(deviceSerial)
	if err != nil {
		log.Printf("Couldn't get device state: %v", err)
		return err
	}

	log.Printf("Requested action: %v, Current state: %v", action, curState)
	if (action == myq.ActionOpen && curState == myq.StateClosed) || (action == myq.ActionClose && curState == myq.StateOpen) {
		log.Printf("Attempting action: %v", action)
		err := s.SetDoorState(deviceSerial, action)
		if err != nil {
			log.Printf("Unable to set door state: %v", err)
			return err
		}
	} else {
		log.Printf("Action and state mismatch: garage state is not valid for executing requested action")
		return nil
	}

	log.Printf("Waiting for door to %s...\n", action)

	var currentState string
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		state, err := s.DeviceState(deviceSerial)
		if err != nil {
			return err
		}
		if state != currentState {
			if currentState != "" {
				log.Printf("Door state changed to %s\n", state)
			}
			currentState = state
		}
		if currentState == desiredState {
			break
		}
		time.Sleep(5 * time.Second)
	}

	if currentState != desiredState {
		return fmt.Errorf("timed out waiting for door to be %s", desiredState)
	}

	return nil
}

// check for env vars and validate that a myq_email and myq_pass exists
func checkEnvVars() {
	// override config with env vars if present
	if value, exists := os.LookupEnv("MYQ_EMAIL"); exists {
		Config.Global.MyQEmail = value
	}
	if value, exists := os.LookupEnv("MYQ_PASS"); exists {
		Config.Global.MyQEmail = value
	}
	if Config.Global.MyQEmail == "" || Config.Global.MyQPass == "" {
		log.Fatal("MYQ_EMAIL and MYQ_PASS must be defined in the config file or as env vars")
	}
}

func withinGeofence(point Point, center Point, radius float64) bool {
	// Calculate the distance between the point and the center of the circle
	distance := distance(point, center)
	return distance <= radius
}

func distance(point1 Point, point2 Point) float64 {
	// Calculate the distance between two points using the haversine formula
	const radius = 6371 // Earth's radius in kilometers
	lat1 := toRadians(point1.Lat)
	lat2 := toRadians(point2.Lat)
	deltaLat := toRadians(point2.Lat - point1.Lat)
	deltaLon := toRadians(point2.Lng - point1.Lng)
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := radius * c
	return d
}

func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// check if outside close geo or inside open geo and set garage door state accordingly
func checkGeoFence(car *Car) {
	if car.OpLock {
		return
	}
	car.OpLock = true
	if car.CurLat == 0 || car.CurLng == 0 {
		car.OpLock = false
		return // need valid lat and lng to check fence
	}

	// Define a point to check
	point := Point{car.CurLat, car.CurLng}

	if carAtHome && !withinGeofence(point, car.GarageCloseGeo.Center, car.GarageCloseGeo.Radius) { // check if outside the close geofence, meaning we should close the door
		setGarageDoor(car.MyQSerial, myq.ActionClose)
		carAtHome = false
		time.Sleep(5 * time.Minute) // keep opLock true for 5 minutes to prevent flapping in case of overlapping geofences
	} else if !carAtHome && withinGeofence(point, car.GarageOpenGeo.Center, car.GarageOpenGeo.Radius) {
		setGarageDoor(car.MyQSerial, myq.ActionOpen)
		carAtHome = true
		time.Sleep(5 * time.Minute) // keep opLock true for 5 minutes to prevent flapping in case of overlapping geofences
	}

	car.OpLock = false
}
