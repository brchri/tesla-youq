package gdo

import (
	"errors"
	"fmt"

	"github.com/brchri/tesla-youq/internal/gdo/mqtt"
	"github.com/brchri/tesla-youq/internal/gdo/ratgdo"
)

type GDO interface {
	SetGarageDoor(action string) (err error)
}

func Initialize(config map[string]interface{}) (GDO, error) {
	typeValue, exists := config["type"]
	if !exists {
		return nil, errors.New("gdo type not defined")
	}

	switch typeValue {
	case "ratgdo":
		return ratgdo.Initialize(config)
	case "mqtt":
		return mqtt.Initialize(config)
	default:
		return nil, fmt.Errorf("gdo type %s not recognized", typeValue)
	}
}
