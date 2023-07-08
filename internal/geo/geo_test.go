package geo

import (
	"fmt"
	util "myq-teslamate-geofence/internal/util"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

type MockMyqSession struct{}
type testParamsStruct struct {
	setUsernameCount  int
	setPasswordCount  int
	loginCount        int
	deviceStateCount  int
	setDoorStateCount int
}

var (
	testParams *testParamsStruct

	deviceStateReturnValue string
	deviceStateReturnError error
	loginError             error
	setDoorStateError      error

	car *util.Car
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
	return setDoorStateError
}

func (m *MockMyqSession) New() {}

func init() {
	util.LoadConfig(filepath.Join("..", "..", "config.example.yml"))
	car = util.Config.Cars[0]
	util.Config.Global.OpCooldown = 0
}

func Test_CheckGeoFence_Leaving(t *testing.T) {
	myqExec = &MockMyqSession{}
	var wg sync.WaitGroup

	// TEST 1 - Leaving home, garage close
	car.AtHome = true
	testParams = &testParamsStruct{}
	car.CurLat = car.GarageCloseGeo.Center.Lat + 10
	car.CurLng = car.GarageCloseGeo.Center.Lng

	deviceStateReturnValue = "open"

	wg.Add(1)
	go func() {
		defer wg.Done()
		CheckGeoFence(util.Config, car)
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

func Test_CheckGeofence_LeaveRetry(t *testing.T) {
	// TEST 2 - Leaving home, garage close, fail and retry 3 times
	car.AtHome = true
	testParams = &testParamsStruct{}
	car.CurLat = car.GarageCloseGeo.Center.Lat + 10
	car.CurLng = car.GarageCloseGeo.Center.Lng

	deviceStateReturnValue = "open"
	setDoorStateError = fmt.Errorf("mock error")

	CheckGeoFence(util.Config, car)

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

func Test_CheckGeofence_Arrive(t *testing.T) {
	// TEST 3 - Arriving Home
	car.AtHome = false
	testParams = &testParamsStruct{}
	car.CurLat = car.GarageCloseGeo.Center.Lat
	car.CurLng = car.GarageCloseGeo.Center.Lng
	var wg sync.WaitGroup

	deviceStateReturnValue = "closed"
	setDoorStateError = nil

	wg.Add(1)
	go func() {
		defer wg.Done()
		CheckGeoFence(util.Config, car)
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
