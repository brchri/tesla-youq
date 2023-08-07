package util

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type (
	Point struct {
		Lat float64 `yaml:"lat"`
		Lng float64 `yaml:"lng"`
	}

	// defines which teslamate defined geofence change will trigger an event, e.g. "home" to "not_home"
	TeslamateGeofenceTrigger struct {
		From string `yaml:"from"`
		To   string `yaml:"to"`
	}

	// defines a center point and two radii (distances) to define open and close geofences
	CircularGeofence struct {
		Center        Point   `yaml:"center"`
		CloseDistance float64 `yaml:"close_distance"` // defines a radius from the center point; when vehicle moves from < distance to > distance, garage will close
		OpenDistance  float64 `yaml:"open_distance"`  // defines a radius from the center point; when vehicle moves from > distance to < distance, garage will open
	}

	// defines triggers for open and close action for teslamate geofences
	TeslamateGeofence struct {
		Close TeslamateGeofenceTrigger `yaml:"close_trigger"` // garage will close when vehicle moves from `from` to `to`
		Open  TeslamateGeofenceTrigger `yaml:"open_trigger"`  // garage will open when vehicle moves from `from` to `to`
	}

	// contains 2 geofences, open and close, each of which are a list of lat/long points defining the polygon
	PolygonGeofence struct {
		Close []Point `yaml:"close"` // list of points defining a polygon; when vehicle moves from inside this geofence to outside, garage will close
		Open  []Point `yaml:"open"`  // list of points defining a polygon; when vehicle moves from outside this geofence to inside, garage will open
	}

	Car struct {
		ID                 int         `yaml:"teslamate_car_id"` // mqtt identifier for vehicle
		GarageDoor         *GarageDoor // bidirectional pointer to GarageDoor containing car
		CurLat             float64     // current latitude
		CurLng             float64     // current longitude
		CurDistance        float64     // current distance from garagedoor location
		PrevGeofence       string      // geofence previously ascribed to car
		CurGeofence        string      // updated geofence ascribed to car when published to mqtt
		InsidePolyOpenGeo  bool        // indicates if car is currently inside the polygon_open_geofence
		InsidePolyCloseGeo bool        // indicates if car is currently inside the polygon_close_geofence
	}

	// defines a garage door with one unique geofence type: circular, teslamate, or polygon
	// only one geofence type may be defined per garage door
	// if more than one defined, priority will be polygon > circular > teslamate
	GarageDoor struct {
		CircularGeofence  CircularGeofence  `yaml:"circular_geofence"`
		TeslamateGeofence TeslamateGeofence `yaml:"teslamate_geofence"`
		PolygonGeofence   PolygonGeofence   `yaml:"polygon_geofence"`
		MyQSerial         string            `yaml:"myq_serial"`
		Cars              []*Car            `yaml:"cars"` // cars housed within this garage
		OpLock            bool              // controls if garagedoor has been operated recently to prevent flapping
		GeofenceType      string            //indicates whether garage door uses teslamate's geofence or not (checked during runtime)
	}

	ConfigStruct struct {
		Global struct {
			MqttHost          string `yaml:"mqtt_host"`
			MqttPort          int    `yaml:"mqtt_port"`
			MqttClientID      string `yaml:"mqtt_client_id"`
			MqttUser          string `yaml:"mqtt_user"`
			MqttPass          string `yaml:"mqtt_pass"`
			MqttUseTls        bool   `yaml:"mqtt_use_tls"`
			MqttSkipTlsVerify bool   `yaml:"mqtt_skip_tls_verify"`
			OpCooldown        int    `yaml:"cooldown"`
			MyQEmail          string `yaml:"myq_email"`
			MyQPass           string `yaml:"myq_pass"`
		} `yaml:"global"`
		GarageDoors []*GarageDoor `yaml:"garage_doors"`
		Testing     bool
	}
)

var Config ConfigStruct

const (
	PolygonGeofenceType   = "PolygonGeofence"   // custom polygon geofence defined by multiple lat/long points
	CircularGeofenceType  = "CircularGeofence"  // circular geofence with center point and radius
	TeslamateGeofenceType = "TeslamateGeofence" // geofence defined in teslamate
)

// checks for valid geofence values for a garage door
// preferred priority is polygon > circular > teslamate
func (g GarageDoor) GetGeofenceType() string {
	if len(g.PolygonGeofence.Open) > 0 && len(g.PolygonGeofence.Close) > 0 {
		return PolygonGeofenceType
	} else if g.CircularGeofence.Center.IsPointDefined() && g.CircularGeofence.OpenDistance > 0 && g.CircularGeofence.CloseDistance > 0 {
		return CircularGeofenceType
	} else if g.TeslamateGeofence.Close.From != "" &&
		g.TeslamateGeofence.Close.To != "" &&
		g.TeslamateGeofence.Open.From != "" &&
		g.TeslamateGeofence.Open.To != "" {
		return TeslamateGeofenceType
	} else {
		return ""
	}
}

func (p Point) IsPointDefined() bool {
	// lat=0 lng=0 are valid coordinates, but they're in the middle of the ocean, so safe to assume these mean undefined
	return p.Lat != 0 && p.Lng != 0
}

// load yaml config
func LoadConfig(configFile string) {
	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Could not read config file: %v", err)
	}

	err = yaml.Unmarshal(yamlFile, &Config)
	if err != nil {
		log.Fatalf("Could not load yaml from config file, received error: %v", err)
	}

	for _, g := range Config.GarageDoors {
		g.GeofenceType = g.GetGeofenceType()
	}

	log.Println("Config loaded successfully")
}
