package geo

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/brchri/tesla-youq/internal/mocks"
	"github.com/joeshaw/myq"
	"github.com/stretchr/testify/assert"
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

func Test_getDistanceChangeAction(t *testing.T) {
	distanceCar.CurDistance = 0
	distanceCar.CurLat = distanceCar.GarageDoor.CircularGeofence.Center.Lat + 10
	distanceCar.CurLng = distanceCar.GarageDoor.CircularGeofence.Center.Lng

	assert.Equal(t, myq.ActionClose, getDistanceChangeAction(util.Config, distanceCar))
	assert.Greater(t, distanceCar.CurDistance, distanceCar.GarageDoor.CircularGeofence.CloseDistance)

	distanceCar.CurLat = distanceCar.GarageDoor.CircularGeofence.Center.Lat

	assert.Equal(t, myq.ActionOpen, getDistanceChangeAction(util.Config, distanceCar))
	assert.Less(t, distanceCar.CurDistance, distanceCar.GarageDoor.CircularGeofence.OpenDistance)
}

func Test_getGeoChangeEventAction(t *testing.T) {
	geofenceCar.PrevGeofence = "home"
	geofenceCar.CurGeofence = "not_home"

	assert.Equal(t, myq.ActionClose, getGeoChangeEventAction(util.Config, geofenceCar))

	geofenceCar.PrevGeofence = "not_home"
	geofenceCar.CurGeofence = "home"

	assert.Equal(t, myq.ActionOpen, getGeoChangeEventAction(util.Config, geofenceCar))
}

func Test_isInsidePolygonGeo(t *testing.T) {
	p := util.Point{
		Lat: 46.19292902096646,
		Lng: -123.79984989897177,
	}

	assert.Equal(t, false, isInsidePolygonGeo(p, polygonCar.GarageDoor.PolygonGeofence.Close))

	p = util.Point{
		Lat: 46.19243683948096,
		Lng: -123.80103692981524,
	}

	assert.Equal(t, true, isInsidePolygonGeo(p, polygonCar.GarageDoor.PolygonGeofence.Open))
}

func Test_getPolygonGeoChangeEventAction(t *testing.T) {
	polygonCar.InsidePolyCloseGeo = true
	polygonCar.InsidePolyOpenGeo = true
	polygonCar.CurLat = 46.19292902096646
	polygonCar.CurLng = -123.79984989897177

	assert.Equal(t, myq.ActionClose, getPolygonGeoChangeEventAction(util.Config, polygonCar))
	assert.Equal(t, false, polygonCar.InsidePolyCloseGeo)
	assert.Equal(t, true, polygonCar.InsidePolyOpenGeo)

	polygonCar.InsidePolyOpenGeo = false
	polygonCar.CurLat = 46.19243683948096
	polygonCar.CurLng = -123.80103692981524

	assert.Equal(t, myq.ActionOpen, getPolygonGeoChangeEventAction(util.Config, polygonCar))
}

func Test_CheckCircularGeofence_Leaving_NotLoggedIn(t *testing.T) {
	myqSession := &mocks.MyqSessionInterface{}
	myqSession.Test(t)
	defer myqSession.AssertExpectations(t)
	myqExec = myqSession

	// TEST 1 - Leaving home, garage close
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return("", errors.New("unauthorized")).Once()
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.EXPECT().New().Once()
	myqSession.EXPECT().Login().Return(nil).Once()
	myqSession.EXPECT().SetUsername(mock.AnythingOfType("string")).Once()
	myqSession.EXPECT().SetPassword(mock.AnythingOfType("string")).Once()
	myqSession.EXPECT().SetDoorState(mock.AnythingOfType("string"), myq.ActionClose).Return(nil).Once()

	distanceCar.CurDistance = 0
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat + 10
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng
	responseChannel := make(chan bool, 1)

	CheckGeofence(util.Config, distanceCar, responseChannel)
	assert.True(t, len(responseChannel) == 0, "Length of responseChannel didn't match the expected length of 0")
}

func Test_CheckCircularGeofence_Leaving_LoggedIn(t *testing.T) {
	myqSession := &mocks.MyqSessionInterface{}
	myqSession.Test(t)
	defer myqSession.AssertExpectations(t)
	myqExec = myqSession

	// TEST 1 - Leaving home, garage close
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.EXPECT().SetDoorState(mock.AnythingOfType("string"), myq.ActionClose).Return(nil).Once()

	distanceCar.CurDistance = 0
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat + 10
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng
	responseChannel := make(chan bool, 1)

	CheckGeofence(util.Config, distanceCar, responseChannel)
	assert.True(t, len(responseChannel) == 0, "Length of responseChannel didn't match the expected length of 0")
}

func Test_CheckCircularGeofence_Arriving_LoggedIn(t *testing.T) {
	myqSession := &mocks.MyqSessionInterface{}
	myqSession.Test(t)
	defer myqSession.AssertExpectations(t)
	myqExec = myqSession

	// TEST 1 - Arriving home, garage open
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	myqSession.EXPECT().SetDoorState(mock.AnythingOfType("string"), myq.ActionOpen).Return(nil).Once()

	distanceCar.CurDistance = 100
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng
	responseChannel := make(chan bool, 1)

	CheckGeofence(util.Config, distanceCar, responseChannel)

	// TEST 2 - Arrived home, garage open, initiating auto-close
	assert.True(t, len(responseChannel) == 1, "Length of responseChannel didn't match the expected length of 1")
	testValidateGeofenceAndClose(t, distanceCar, myqSession, responseChannel)
}

func Test_CheckCircularGeofence_Arriving_LoggedIn_Retry(t *testing.T) {
	myqSession := &mocks.MyqSessionInterface{}
	myqSession.Test(t)
	defer myqSession.AssertExpectations(t)
	myqExec = myqSession

	// TEST 1 - Arriving home, garage open
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Times(3)
	myqSession.EXPECT().SetDoorState(mock.AnythingOfType("string"), myq.ActionOpen).Return(errors.New("some error")).Twice()
	myqSession.EXPECT().SetDoorState(mock.AnythingOfType("string"), myq.ActionOpen).Return(nil).Once()
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()

	distanceCar.CurDistance = 100
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng
	responseChannel := make(chan bool, 1)

	CheckGeofence(util.Config, distanceCar, responseChannel)

	// TEST 2 - Arrived home, garage open, initiating auto-close
	assert.True(t, len(responseChannel) == 1, "Length of responseChannel didn't match the expected length of 1")
	testValidateGeofenceAndClose(t, distanceCar, myqSession, responseChannel)
}

func Test_CheckCircularGeofence_LeaveThenArrive_NotLoggedIn(t *testing.T) {
	myqSession := &mocks.MyqSessionInterface{}
	myqSession.Test(t)
	defer myqSession.AssertExpectations(t)
	myqExec = myqSession

	// TEST 1 - Leaving home, garage close
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return("", errors.New("unauthorized")).Once()
	myqSession.EXPECT().New().Once()
	myqSession.EXPECT().SetUsername(mock.AnythingOfType("string")).Once()
	myqSession.EXPECT().SetPassword(mock.AnythingOfType("string")).Once()
	myqSession.EXPECT().Login().Return(nil).Once()
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	myqSession.EXPECT().SetDoorState(mock.AnythingOfType("string"), myq.ActionClose).Return(nil).Once()
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()

	distanceCar.CurDistance = 0
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat + 10
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng

	responseChannel := make(chan bool, 1)

	CheckGeofence(util.Config, distanceCar, responseChannel)

	myqSession.AssertExpectations(t) // midpoint check

	// TEST 2 - Arriving home, garage open
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.EXPECT().SetDoorState(mock.AnythingOfType("string"), myq.ActionOpen).Return(nil).Once()
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	distanceCar.CurLat = distanceGarageDoor.CircularGeofence.Center.Lat
	distanceCar.CurLng = distanceGarageDoor.CircularGeofence.Center.Lng

	CheckGeofence(util.Config, distanceCar, responseChannel)

	// TEST 3 - Arrived home, garage open, initiating auto-close
	assert.True(t, len(responseChannel) == 1, "Length of responseChannel didn't match the expected length of 1")
	testValidateGeofenceAndClose(t, distanceCar, myqSession, responseChannel)
}

func testValidateGeofenceAndClose(t *testing.T, car *util.Car, myqSession *mocks.MyqSessionInterface, responseChannel chan bool) {
	if <-responseChannel {
		myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
		myqSession.EXPECT().SetDoorState(mock.AnythingOfType("string"), myq.ActionClose).Return(nil).Once()
		myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
		VailidateGeofenceAndClose(util.Config, car, myq.ActionClose)
	} else {
		t.Error("Expected the value in responseChannel to be true")
	}
}

func Test_CheckTeslamateGeofence_Leaving_LoggedIn(t *testing.T) {
	myqSession := &mocks.MyqSessionInterface{}
	myqSession.Test(t)
	defer myqSession.AssertExpectations(t)
	myqExec = myqSession

	// TEST 1 - Leaving home, garage close
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	myqSession.EXPECT().SetDoorState(mock.Anything, myq.ActionClose).Return(nil).Once()
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()

	geofenceCar.PrevGeofence = "home"
	geofenceCar.CurGeofence = "not_home"
	responseChannel := make(chan bool, 1)

	CheckGeofence(util.Config, geofenceCar, responseChannel)

	assert.True(t, len(responseChannel) == 0, "Length of responseChannel didn't match the expected length of 0")
}

func Test_CheckTeslamateGeofence_Arriving_LoggedIn(t *testing.T) {
	myqSession := &mocks.MyqSessionInterface{}
	myqSession.Test(t)
	defer myqSession.AssertExpectations(t)
	myqExec = myqSession

	// TEST 1 - Leaving home, garage close
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.EXPECT().SetDoorState(mock.Anything, myq.ActionOpen).Return(nil).Once()
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()

	geofenceCar.PrevGeofence = "not_home"
	geofenceCar.CurGeofence = "home"
	responseChannel := make(chan bool, 1)

	CheckGeofence(util.Config, geofenceCar, responseChannel)

	// TEST 2 - Arrived home, garage open, initiating auto-close
	assert.True(t, len(responseChannel) == 1, "Length of responseChannel didn't match the expected length of 1")
	testValidateGeofenceAndClose(t, geofenceCar, myqSession, responseChannel)
}

func Test_CheckPolyGeofence_Leaving_NotLoggedIn(t *testing.T) {
	myqSession := &mocks.MyqSessionInterface{}
	myqSession.Test(t)
	defer myqSession.AssertExpectations(t)
	myqExec = myqSession

	// TEST 1 - Leaving home, garage close
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return("", errors.New("unauthorized")).Once()
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.EXPECT().New().Once()
	myqSession.EXPECT().Login().Return(nil).Once()
	myqSession.EXPECT().SetUsername(mock.AnythingOfType("string")).Once()
	myqSession.EXPECT().SetPassword(mock.AnythingOfType("string")).Once()
	myqSession.EXPECT().SetDoorState(mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Once()

	polygonCar.InsidePolyCloseGeo = true
	polygonCar.InsidePolyOpenGeo = true
	polygonCar.CurLat = 46.19292902096646
	polygonCar.CurLng = -123.79984989897177
	responseChannel := make(chan bool, 1)

	CheckGeofence(util.Config, polygonCar, responseChannel)
	assert.True(t, len(responseChannel) == 0, "Length of responseChannel didn't match the expected length of 0")
}

func Test_CheckPolyGeofence_Arriving_LoggedIn(t *testing.T) {
	myqSession := &mocks.MyqSessionInterface{}
	myqSession.Test(t)
	defer myqSession.AssertExpectations(t)
	myqExec = myqSession

	// TEST 1 - Arriving home, garage open
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateClosed, nil).Once()
	myqSession.EXPECT().SetDoorState(mock.AnythingOfType("string"), myq.ActionOpen).Return(nil).Once()
	myqSession.EXPECT().DeviceState(mock.AnythingOfType("string")).Return(myq.StateOpen, nil).Once()

	polygonCar.InsidePolyCloseGeo = false
	polygonCar.InsidePolyOpenGeo = false
	polygonCar.CurLat = 46.19243683948096
	polygonCar.CurLng = -123.80103692981524
	responseChannel := make(chan bool, 1)

	CheckGeofence(util.Config, polygonCar, responseChannel)

	// TEST 2 - Arrived home, garage open, initiating auto-close
	assert.True(t, len(responseChannel) == 1, "Length of responseChannel didn't match the expected length of 1")
	testValidateGeofenceAndClose(t, polygonCar, myqSession, responseChannel)
}
