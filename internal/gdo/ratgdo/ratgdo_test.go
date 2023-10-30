package ratgdo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var sampleYaml = map[string]interface{}{
	"mqtt_settings": map[string]interface{}{
		"connection": map[string]interface{}{
			"host":            "localhost",
			"port":            1883,
			"client_id":       "test-mqtt-module",
			"user":            "test-user",
			"pass":            "test-pass",
			"use_tls":         false,
			"skip_tls_verify": false,
		},
		"topic_prefix": "home/garage/Main",
	},
}

// Since ratgdo is just a wrapper for mqttGdo with some predefined configs,
// just need to ensure NewRatgdo doesn't throw any errors when returning
// an MqttGdo object
func Test_NewRatgdo(t *testing.T) {
	_, err := NewRatgdo(sampleYaml)
	assert.Equal(t, nil, err)
}
