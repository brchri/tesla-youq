package geo

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	util "github.com/brchri/tesla-youq/internal/util"
)

type MockMyqSession struct{}
type testParamsStruct struct {
	setUsernameCount  int
	setPasswordCount  int
	loginCount        int
	deviceStateCount  int
	setDoorStateCount int
	openActionCount   int
	closeActionCount  int
}

var (
	testParams *testParamsStruct

	deviceStateReturnValue string
	deviceStateReturnError error
	loginError             error
	setDoorStateError      error

	distanceCar        *util.Car
	distanceGarageDoor *util.GarageDoor

	geofenceGarageDoor *util.GarageDoor
	geofenceCar        *util.Car

	polygonGarageDoor *util.GarageDoor
	polygonCar        *util.Car
)

func (m *MockMyqSession) SetUsername(s string) {
	testParams.setUsernameCount++
}

func (m *MockMyqSession) SetPassword(s string) {
	testParams.setPasswordCount++
}

func (m *MockMyqSession) DeviceState(s string) (string, error) {
	testParams.deviceStateCount++
	return deviceStateReturnValue, deviceStateReturnError
}

func (m *MockMyqSession) Login() error {
	testParams.loginCount++
	return loginError
}

func (m *MockMyqSession) SetDoorState(serialNumber, action string) error {
	testParams.setDoorStateCount++
	if action == "open" {
		testParams.openActionCount++
	} else if action == "close" {
		testParams.closeActionCount++
	}
	return setDoorStateError
}

func (m *MockMyqSession) New() {}

func init() {
	util.LoadConfig(filepath.Join("..", "..", "config.example.yml"))

	// used for testing events based on distance
	distanceGarageDoor = util.Config.GarageDoors[0]
	distanceCar = distanceGarageDoor.Cars[0]
	distanceCar.GarageDoor = distanceGarageDoor
	distanceCar.GarageDoor.GeofenceType = util.CircularGeofenceType

	// used for testing events based on teslamate geofence changes
	geofenceGarageDoor = util.Config.GarageDoors[1]
	geofenceCar = geofenceGarageDoor.Cars[0]
	geofenceCar.GarageDoor = geofenceGarageDoor
	geofenceCar.GarageDoor.GeofenceType = util.TeslamateGeofenceType

	// used for testing events based on teslamate geofence changes
	polygonGarageDoor = util.Config.GarageDoors[2]
	polygonCar = polygonGarageDoor.Cars[0]
	polygonCar.GarageDoor = polygonGarageDoor
	polygonCar.GarageDoor.GeofenceType = util.PolygonGeofenceType

	util.Config.Global.OpCooldown = 0
	myqExec = &MockMyqSession{}
}

func Test_CheckGeoFence_DistanceTrigger_Leaving(t *testing.T) {
	var wg sync.WaitGroup

	// TEST 1 - Leaving home, garage close
	distanceCar.CurDistance = 0
	testParams = &testParamsStruct{}
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat + 10
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng

	deviceStateReturnValue = "open"

	wg.Add(1)
	go func() {
		defer wg.Done()
		CheckGeoFence(util.Config, distanceCar)
	}()
	// wait for SetGarageDoor call and then update call
	for {
		if testParams.setDoorStateCount > 0 {
			deviceStateReturnValue = "closed"
			break
		}
	}
	wg.Wait()
	want := []int{1, 1, 1, 1}
	got := []int{testParams.setUsernameCount,
		testParams.setPasswordCount,
		testParams.loginCount,
		testParams.setDoorStateCount,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("Test 1 failed, got %v, want %v", got, want)
	}
}

func Test_CheckGeofence_DistanceTrigger_LeaveRetry(t *testing.T) {
	// TEST 2 - Leaving home, garage close, fail and retry 3 times
	distanceCar.CurDistance = 0
	testParams = &testParamsStruct{}
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat + 10
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng

	deviceStateReturnValue = "open"
	setDoorStateError = fmt.Errorf("mock error")

	CheckGeoFence(util.Config, distanceCar)

	want := []int{3, 3, 3, 3}
	got := []int{testParams.setUsernameCount,
		testParams.setPasswordCount,
		testParams.loginCount,
		testParams.setDoorStateCount,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("Test 2 failed, got %v, want %v", got, want)
	}
}

func Test_CheckGeofence_DistanceTrigger_Arrive(t *testing.T) {
	// TEST 3 - Arriving Home
	distanceCar.CurDistance = 1
	testParams = &testParamsStruct{}
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng
	var wg sync.WaitGroup

	deviceStateReturnValue = "closed"
	setDoorStateError = nil

	wg.Add(1)
	go func() {
		defer wg.Done()
		CheckGeoFence(util.Config, distanceCar)
	}()
	// wait for SetGarageDoor call and then update call
	for {
		if testParams.setDoorStateCount > 0 {
			deviceStateReturnValue = "open"
			break
		}
	}
	wg.Wait()

	want := []int{1, 1, 1, 1}
	got := []int{testParams.setUsernameCount,
		testParams.setPasswordCount,
		testParams.loginCount,
		testParams.setDoorStateCount,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("Test 3 failed, got %v, want %v", got, want)
	}
}

func Test_CheckGeofence_DistanceTrigger_Leave_Then_Arrive(t *testing.T) {
	distanceCar.CurDistance = 0
	distanceCar.CurLat = distanceCar.GarageDoor.CircularGeofence.Center.Lat + 1
	distanceCar.CurLng = distanceCar.GarageDoor.CircularGeofence.Center.Lng
	testParams = &testParamsStruct{}
	var wg sync.WaitGroup

	deviceStateReturnValue = "open"
	deviceStateReturnError = nil
	setDoorStateError = nil

	wg.Add(1)
	go func() {
		defer wg.Done()
		CheckGeoFence(util.Config, distanceCar)
	}()
	// wait for SetGarageDoor call and then update call
	for {
		if testParams.setDoorStateCount > 0 {
			deviceStateReturnValue = "closed"
			break
		}
	}
	wg.Wait()

	// check garage would've been closed
	want := []int{1, 1, 1, 1, 1}
	got := []int{testParams.setUsernameCount,
		testParams.setPasswordCount,
		testParams.loginCount,
		testParams.setDoorStateCount,
		testParams.closeActionCount,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("leave function call counts failed, got %v, want %v", got, want)
	}

	distanceCar.CurLat = distanceCar.CurLat + 1
	prevDistance := distanceCar.CurDistance
	CheckGeoFence(util.Config, distanceCar) // should return no-op but will update geofenceCar.PrevGeofence
	if prevDistance == distanceCar.CurDistance {
		t.Errorf("update CurDistance failed")
	}

	distanceCar.CurLat = distanceCar.GarageDoor.CircularGeofence.Center.Lat
	wg.Add(1)
	go func() {
		defer wg.Done()
		CheckGeoFence(util.Config, distanceCar)
	}()
	// wait for SetGarageDoor call and then update call
	for {
		if testParams.setDoorStateCount > 1 {
			deviceStateReturnValue = "open"
			break
		}
	}
	wg.Wait()

	// check garage would've been opened
	want = []int{2, 2, 2, 2, 1}
	got = []int{testParams.setUsernameCount,
		testParams.setPasswordCount,
		testParams.loginCount,
		testParams.setDoorStateCount,
		testParams.openActionCount,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("arrive function call counts failed, got %v, want %v", got, want)
	}
}

func Test_CheckGeoFence_GeofenceTrigger_Leaving(t *testing.T) {
	var wg sync.WaitGroup

	// TEST 1 - Leaving home, garage close
	geofenceCar.PrevGeofence = "home"
	geofenceCar.CurGeofence = "not_home"
	testParams = &testParamsStruct{}

	deviceStateReturnValue = "open"

	wg.Add(1)
	go func() {
		defer wg.Done()
		CheckGeoFence(util.Config, geofenceCar)
	}()
	// wait for SetGarageDoor call and then update call
	for {
		if testParams.setDoorStateCount > 0 {
			deviceStateReturnValue = "closed"
			break
		}
	}
	wg.Wait()
	want := []int{1, 1, 1, 1}
	got := []int{testParams.setUsernameCount,
		testParams.setPasswordCount,
		testParams.loginCount,
		testParams.setDoorStateCount,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("Test 1 failed, got %v, want %v", got, want)
	}
}

func Test_CheckGeofence_GeofenceTrigger_Arrive(t *testing.T) {
	geofenceCar.PrevGeofence = "not_home"
	geofenceCar.CurGeofence = "home"
	testParams = &testParamsStruct{}
	var wg sync.WaitGroup

	deviceStateReturnValue = "closed"
	deviceStateReturnError = nil
	setDoorStateError = nil

	wg.Add(1)
	go func() {
		defer wg.Done()
		CheckGeoFence(util.Config, geofenceCar)
	}()
	// wait for SetGarageDoor call and then update call
	for {
		if testParams.setDoorStateCount > 0 {
			deviceStateReturnValue = "open"
			break
		}
	}
	wg.Wait()

	want := []int{1, 1, 1, 1}
	got := []int{testParams.setUsernameCount,
		testParams.setPasswordCount,
		testParams.loginCount,
		testParams.setDoorStateCount,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("Test 3 failed, got %v, want %v", got, want)
	}
}

func Test_CheckGeofence_GeofenceTrigger_Leave_Then_Arrive(t *testing.T) {
	// car is leaving
	geofenceCar.PrevGeofence = "home"
	geofenceCar.CurGeofence = "not_home"
	testParams = &testParamsStruct{}
	var wg sync.WaitGroup

	deviceStateReturnValue = "open"
	deviceStateReturnError = nil
	setDoorStateError = nil

	wg.Add(1)
	go func() {
		defer wg.Done()
		CheckGeoFence(util.Config, geofenceCar)
	}()
	// wait for SetGarageDoor call and then update call
	for {
		if testParams.setDoorStateCount > 0 {
			deviceStateReturnValue = "closed"
			break
		}
	}
	wg.Wait()

	// check garage would've been closed
	want := []int{1, 1, 1, 1, 1}
	got := []int{testParams.setUsernameCount,
		testParams.setPasswordCount,
		testParams.loginCount,
		testParams.setDoorStateCount,
		testParams.closeActionCount,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("leave function call counts failed, got %v, want %v", got, want)
	}

	// car has moved far away, outside of all defined geofences
	geofenceCar.PrevGeofence = geofenceCar.CurGeofence
	geofenceCar.CurGeofence = "not_home"
	CheckGeoFence(util.Config, geofenceCar)

	// check no new ops on garage
	want = []int{1, 1, 1, 1, 1}
	got = []int{testParams.setUsernameCount,
		testParams.setPasswordCount,
		testParams.loginCount,
		testParams.setDoorStateCount,
		testParams.closeActionCount,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("leave function call counts failed, got %v, want %v", got, want)
	}

	// car is now close to home
	geofenceCar.PrevGeofence = geofenceCar.CurGeofence
	geofenceCar.CurGeofence = "home"
	wg.Add(1)
	go func() {
		defer wg.Done()
		CheckGeoFence(util.Config, geofenceCar)
	}()
	// wait for SetGarageDoor call and then update call
	for {
		if testParams.setDoorStateCount > 1 {
			deviceStateReturnValue = "open"
			break
		}
	}
	wg.Wait()

	// check garage would've been opened
	want = []int{2, 2, 2, 2, 1}
	got = []int{testParams.setUsernameCount,
		testParams.setPasswordCount,
		testParams.loginCount,
		testParams.setDoorStateCount,
		testParams.openActionCount,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("arrive function call counts failed, got %v, want %v", got, want)
	}

	// car is home
	geofenceCar.PrevGeofence = geofenceCar.CurGeofence
	geofenceCar.CurGeofence = "home"
	CheckGeoFence(util.Config, geofenceCar)

	// check no new ops on garage
	want = []int{2, 2, 2, 2, 1}
	got = []int{testParams.setUsernameCount,
		testParams.setPasswordCount,
		testParams.loginCount,
		testParams.setDoorStateCount,
		testParams.closeActionCount,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("leave function call counts failed, got %v, want %v", got, want)
	}
}

func Test_CheckPolyGeoFence_GeofenceTrigger_Leaving(t *testing.T) {
	var wg sync.WaitGroup

	// Leaving home, garage close
	polygonCar.InsideCloseGeo = true
	polygonCar.InsideOpenGeo = true
	polygonCar.CurLat = 46.19292902096646
	polygonCar.CurLng = -123.79984989897177
	testParams = &testParamsStruct{}

	deviceStateReturnValue = "open"

	wg.Add(1)
	go func() {
		defer wg.Done()
		CheckGeoFence(util.Config, polygonCar)
	}()
	// wait for SetGarageDoor call and then update call
	for {
		if testParams.setDoorStateCount > 0 {
			deviceStateReturnValue = "closed"
			break
		}
	}
	wg.Wait()
	want := []int{1, 1, 1, 1}
	got := []int{testParams.setUsernameCount,
		testParams.setPasswordCount,
		testParams.loginCount,
		testParams.setDoorStateCount,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("Test 1 failed, got %v, want %v", got, want)
	}
}

func Test_CheckPolyGeoFence_GeofenceTrigger_Arriving(t *testing.T) {
	var wg sync.WaitGroup

	// Arriving home, garage open
	polygonCar.InsideCloseGeo = false
	polygonCar.InsideOpenGeo = false
	polygonCar.CurLat = 46.19243683948096
	polygonCar.CurLng = -123.80103692981524
	testParams = &testParamsStruct{}

	deviceStateReturnValue = "closed"

	wg.Add(1)
	go func() {
		defer wg.Done()
		CheckGeoFence(util.Config, polygonCar)
	}()
	// wait for SetGarageDoor call and then update call
	for {
		if testParams.setDoorStateCount > 0 {
			deviceStateReturnValue = "open"
			break
		}
	}
	wg.Wait()
	want := []int{1, 1, 1, 1}
	got := []int{testParams.setUsernameCount,
		testParams.setPasswordCount,
		testParams.loginCount,
		testParams.setDoorStateCount,
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("Test 1 failed, got %v, want %v", got, want)
	}
}
