package types

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
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
		CarAtHome      bool
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
