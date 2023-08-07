package geo

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/brchri/tesla-youq/internal/mocks"
	"github.com/joeshaw/myq"
	"github.com/stretchr/testify/mock"

	util "github.com/brchri/tesla-youq/internal/util"
)

var (
	distanceCar        *util.Car
	distanceGarageDoor *util.GarageDoor

	geofenceGarageDoor *util.GarageDoor
	geofenceCar        *util.Car

	polygonGarageDoor *util.GarageDoor
	polygonCar        *util.Car
)

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
}

func Test_CheckCircularGeofence_Leaving_NotLoggedIn(t *testing.T) {
	myqSession := mocks.NewMyqSessionInterface(t)
	myqExec = myqSession

	// TEST 1 - Leaving home, garage close
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return("", errors.New("unauthorized")).Once()
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.On("New").Once()
	myqSession.On("Login").Return(nil).Once()
	myqSession.On("SetUsername", mock.AnythingOfType("string")).Once()
	myqSession.On("SetPassword", mock.AnythingOfType("string")).Once()
	myqSession.On("SetDoorState", mock.AnythingOfType("string"), myq.ActionClose).Return(nil).Once()

	distanceCar.CurDistance = 0
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat + 10
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng

	CheckGeofence(util.Config, distanceCar)

	myqSession.AssertExpectations(t)
	// mockery command to generate interface: .\mockery.exe --dir internal\geo --name "MyqSessionInterface"
}

func Test_CheckCircularGeofence_Leaving_LoggedIn(t *testing.T) {
	myqSession := mocks.NewMyqSessionInterface(t)
	myqExec = myqSession

	// TEST 1 - Leaving home, garage close
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.On("SetDoorState", mock.AnythingOfType("string"), myq.ActionClose).Return(nil).Once()

	distanceCar.CurDistance = 0
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat + 10
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng

	CheckGeofence(util.Config, distanceCar)

	myqSession.AssertExpectations(t)
	// mockery command to generate interface: .\mockery.exe --dir internal\geo --name "MyqSessionInterface"
}

func Test_CheckCircularGeofence_Arriving_LoggedIn(t *testing.T) {
	myqSession := mocks.NewMyqSessionInterface(t)
	myqExec = myqSession

	// TEST 1 - Arriving home, garage open
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	myqSession.On("SetDoorState", mock.AnythingOfType("string"), myq.ActionOpen).Return(nil).Once()

	distanceCar.CurDistance = 100
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng

	CheckGeofence(util.Config, distanceCar)

	myqSession.AssertExpectations(t)
	// mockery command to generate interface: .\mockery.exe --dir internal\geo --name "MyqSessionInterface"
}

func Test_CheckCircularGeofence_Arriving_LoggedIn_Retry(t *testing.T) {
	myqSession := mocks.NewMyqSessionInterface(t)
	myqExec = myqSession

	// TEST 1 - Arriving home, garage open
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Times(3)
	myqSession.On("SetDoorState", mock.AnythingOfType("string"), myq.ActionOpen).Return(errors.New("some error")).Twice()
	myqSession.On("SetDoorState", mock.AnythingOfType("string"), myq.ActionOpen).Return(nil).Once()
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()

	distanceCar.CurDistance = 100
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng

	CheckGeofence(util.Config, distanceCar)

	myqSession.AssertExpectations(t)
	// mockery command to generate interface: .\mockery.exe --dir internal\geo --name "MyqSessionInterface"
}

func Test_CheckCircularGeofence_LeaveThenArrive_NotLoggedIn(t *testing.T) {
	myqSession := mocks.NewMyqSessionInterface(t)
	myqExec = myqSession

	// TEST 1 - Leaving home, garage close
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return("", errors.New("unauthorized")).Once()
	myqSession.On("New").Once()
	myqSession.On("SetUsername", mock.AnythingOfType("string")).Once()
	myqSession.On("SetPassword", mock.AnythingOfType("string")).Once()
	myqSession.On("Login").Return(nil).Once()
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	myqSession.On("SetDoorState", mock.AnythingOfType("string"), myq.ActionClose).Return(nil).Once()
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()

	distanceCar.CurDistance = 0
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat + 10
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng

	CheckGeofence(util.Config, distanceCar)

	myqSession.AssertExpectations(t)

	// TEST 2 - Arriving home, garage open
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.On("SetDoorState", mock.AnythingOfType("string"), myq.ActionOpen).Return(nil).Once()
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng

	CheckGeofence(util.Config, distanceCar)

	myqSession.AssertExpectations(t)
	// mockery command to generate interface: .\mockery.exe --dir internal\geo --name "MyqSessionInterface"
}

func Test_CheckTeslamateGeofence_Leaving_LoggedIn(t *testing.T) {
	myqSession := mocks.NewMyqSessionInterface(t)
	myqExec = myqSession

	// TEST 1 - Leaving home, garage close
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	myqSession.On("SetDoorState", mock.Anything, myq.ActionClose).Return(nil).Once()
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()

	geofenceCar.PrevGeofence = "home"
	geofenceCar.CurGeofence = "not_home"

	CheckGeofence(util.Config, geofenceCar)

	myqSession.AssertExpectations(t)
}

func Test_CheckTeslamateGeofence_Arriving_LoggedIn(t *testing.T) {
	myqSession := mocks.NewMyqSessionInterface(t)
	myqExec = myqSession

	// TEST 1 - Leaving home, garage close
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.On("SetDoorState", mock.Anything, myq.ActionOpen).Return(nil).Once()
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()

	geofenceCar.PrevGeofence = "not_home"
	geofenceCar.CurGeofence = "home"

	CheckGeofence(util.Config, geofenceCar)

	myqSession.AssertExpectations(t)
}

func Test_CheckPolyGeofence_Leaving_NotLoggedIn(t *testing.T) {
	myqSession := mocks.NewMyqSessionInterface(t)
	myqExec = myqSession

	// TEST 1 - Leaving home, garage close
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return("", errors.New("unauthorized")).Once()
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.On("New").Once()
	myqSession.On("Login").Return(nil).Once()
	myqSession.On("SetUsername", mock.AnythingOfType("string")).Once()
	myqSession.On("SetPassword", mock.AnythingOfType("string")).Once()
	myqSession.On("SetDoorState", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Once()

	polygonCar.InsidePolyCloseGeo = true
	polygonCar.InsidePolyOpenGeo = true
	polygonCar.CurLat = 46.19292902096646
	polygonCar.CurLng = -123.79984989897177

	CheckGeofence(util.Config, polygonCar)

	myqSession.AssertExpectations(t)
}

func Test_CheckPolyGeofence_Arriving_LoggedIn(t *testing.T) {
	myqSession := mocks.NewMyqSessionInterface(t)
	myqExec = myqSession

	// TEST 1 - Arriving home, garage open
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.On("SetDoorState", mock.AnythingOfType("string"), myq.ActionOpen).Return(nil).Once()
	myqSession.On("DeviceState", mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()

	polygonCar.InsidePolyCloseGeo = false
	polygonCar.InsidePolyOpenGeo = false
	polygonCar.CurLat = 46.19243683948096
	polygonCar.CurLng = -123.80103692981524

	CheckGeofence(util.Config, polygonCar)

	myqSession.AssertExpectations(t)
}
