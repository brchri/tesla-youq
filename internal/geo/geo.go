package geo

import (
	"fmt"
	"log"
	"math"
	"os"
	"time"

	util "github.com/brchri/tesla-youq/internal/util"

	"github.com/joeshaw/myq"
)

// interface that allows api calls to myq to be abstracted and mocked by testing functions
type MyqSessionInterface interface {
	DeviceState(serialNumber string) (string, error)
	Login() error
	SetDoorState(serialNumber, action string) error
	SetUsername(string)
	SetPassword(string)
	New()
}

// implements MyqSessionInterface interface but is only a wrapper for the actual myq package
type MyqSessionWrapper struct {
	myqSession *myq.Session
}

func (m *MyqSessionWrapper) SetUsername(s string) {
	m.myqSession.Username = s
}

func (m *MyqSessionWrapper) SetPassword(s string) {
	m.myqSession.Password = s
}

func (m *MyqSessionWrapper) DeviceState(s string) (string, error) {
	return m.myqSession.DeviceState(s)
}

func (m *MyqSessionWrapper) Login() error {
	return m.myqSession.Login()
}

func (m *MyqSessionWrapper) SetDoorState(serialNumber, action string) error {
	return m.myqSession.SetDoorState(serialNumber, action)
}

func (m *MyqSessionWrapper) New() {
	m.myqSession = &myq.Session{}
}

var myqExec MyqSessionInterface = &MyqSessionWrapper{} // executes myq package commands

func distance(point1 util.Point, point2 util.Point) float64 {
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
func CheckGeoFence(config util.ConfigStruct, car *util.Car) {

	if car.CurLat == 0 || car.CurLng == 0 {
		return // need valid lat and lng to check fence
	}

	// Define a carLocation to check
	carLocation := util.Point{
		Lat: car.CurLat,
		Lng: car.CurLng,
	}

	// update car's current distance, and store the previous distance in a variable
	prevDistance := car.CurDistance
	car.CurDistance = distance(carLocation, car.GarageDoor.Location)

	// check if car has crossed a geofence and set an appropriate action
	var action string
	if prevDistance <= car.GarageDoor.CloseRadius && car.CurDistance > car.GarageDoor.CloseRadius { // car was within close geofence, but now beyond it (car left geofence)
		action = myq.ActionClose
	} else if prevDistance >= car.GarageDoor.OpenRadius && car.CurDistance < car.GarageDoor.OpenRadius { // car was outside of open geofence, but is now within it (car entered geofence)
		action = myq.ActionOpen
	}

	if action == "" || car.GarageDoor.OpLock {
		return // only execute if there's a valid action to execute and the garage door isn't on cooldown
	}

	car.GarageDoor.OpLock = true // set lock so no other threads try to operate the garage before the cooldown period is complete
	log.Printf("Attempting to %s garage door for car %d", action, car.ID)

	// create retry loop to set the garage door state
	for i := 3; i > 0; i-- {
		if err := setGarageDoor(config, car.GarageDoor.MyQSerial, action); err == nil {
			// no error received, so breaking retry loop
			break
		}
		if i == 1 {
			log.Println("Unable to set garage door state, no further attempts will be made")
		} else {
			log.Printf("Retrying set garage door state %d more time(s)", i-1)
		}
	}

	time.Sleep(time.Duration(config.Global.OpCooldown) * time.Minute) // keep opLock true for OpCooldown minutes to prevent flapping in case of overlapping geofences
	car.GarageDoor.OpLock = false                                     // release garage door's operation lock
}

func setGarageDoor(config util.ConfigStruct, deviceSerial string, action string) error {
	s := myqExec
	s.New()
	s.SetUsername(config.Global.MyQEmail)
	s.SetPassword(config.Global.MyQPass)

	var desiredState string
	switch action {
	case myq.ActionOpen:
		desiredState = myq.StateOpen
	case myq.ActionClose:
		desiredState = myq.StateClosed
	}

	if config.Testing {
		log.Printf("TESTING flag set - Would attempt action %v", action)
		return nil
	}

	log.Println("Acquiring MyQ session...")
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

func GetGarageDoorSerials(config util.ConfigStruct) error {
	s := &myq.Session{}
	s.Username = config.Global.MyQEmail
	s.Password = config.Global.MyQPass

	log.Println("Acquiring MyQ session...")
	if err := s.Login(); err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("ERROR: %v\n", err)
		log.SetOutput(os.Stdout)
		return err
	}
	log.Println("Session acquired...")

	devices, err := s.Devices()
	if err != nil {
		log.Printf("Could not get devices: %v", err)
		return err
	}
	for _, d := range devices {
		log.Printf("Device Name: %v", d.Name)
		log.Printf("Device State: %v", d.DoorState)
		log.Printf("Device Type: %v", d.Type)
		log.Printf("Device Serial: %v", d.SerialNumber)
		fmt.Println()
	}

	return nil
}
