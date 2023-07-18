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

	// defines which geofence change will trigger an event, e.g. "home" to "not_home"
	GeofenceTrigger struct {
		From string `yaml:"from"`
		To   string `yaml:"to"`
	}

	Car struct {
		ID           int         `yaml:"teslamate_car_id"` // mqtt identifier for vehicle
		GarageDoor   *GarageDoor // bidirectional pointer to GarageDoor containing car
		CurLat       float64     // current latitude
		CurLng       float64     // current longitude
		CurDistance  float64     // current distance from garagedoor location
		PrevGeofence string      // geofence previously ascribed to car
		CurGeofence  string      // updated geofence ascribed to car when published to mqtt
	}

	// defines a garage door with either a location and open/close radii, OR trigger open/close geofences
	// {Location,CloseRadius,OpenRadius} set is mutually exclusive with {TriggerCloseGeofence,TriggerOpenGeofence}
	// with preference for {TriggerCloseGeofence,TriggerOpenGeofence} set if both are defined
	GarageDoor struct {
		Location             Point           `yaml:"location"`
		CloseRadius          float64         `yaml:"close_radius"`           // distance when leaving to trigger close event
		OpenRadius           float64         `yaml:"open_radius"`            // distance when arriving to trigger open event
		TriggerCloseGeofence GeofenceTrigger `yaml:"trigger_close_geofence"` // geofence cross event to trigger close
		TriggerOpenGeofence  GeofenceTrigger `yaml:"trigger_open_geofence"`  // geofence cross event to trigger open
		MyQSerial            string          `yaml:"myq_serial"`
		Cars                 []*Car          `yaml:"cars"` // cars housed within this garage
		OpLock               bool            // controls if garagedoor has been operated recently to prevent flapping
		UseTeslmateGeofence  bool            //indicates whether garage door uses teslamate's geofence or not (checked during runtime)
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

func (g GeofenceTrigger) IsGeofenceDefined() bool {
	return g.From != "" && g.To != ""
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
	log.Println("Config loaded successfully")
}
