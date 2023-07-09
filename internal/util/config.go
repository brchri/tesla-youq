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

	Car struct {
		ID          int `yaml:"teslamate_car_id"`
		GarageDoor  *GarageDoor
		CurLat      float64
		CurLng      float64
		CurDistance float64
		AtHome      bool
	}

	GarageDoor struct {
		Location    Point   `yaml:"location"`
		CloseRadius float64 `yaml:"close_radius"`
		OpenRadius  float64 `yaml:"open_radius"`
		MyQSerial   string  `yaml:"myq_serial"`
		Cars        []*Car  `yaml:"cars"`
		OpLock      bool
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
		GarageDoors []*GarageDoor `yaml:"garage_doors"`
		Testing     bool
	}
)

var Config ConfigStruct

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
