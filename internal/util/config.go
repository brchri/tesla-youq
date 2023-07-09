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
		CarID             int     `yaml:"teslamate_car_id"`
		MyQSerial         string  `yaml:"myq_serial"`
		GarageLocation    Point   `yaml:"garage_location"`
		GarageCloseRadius float64 `yaml:"garage_close_radius"`
		GarageOpenRadius  float64 `yaml:"garage_open_radius"`
		CurLat            float64
		CurLng            float64
		OpLock            bool
		AtHome            bool
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
		Cars    []*Car `yaml:"cars"`
		Testing bool
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
