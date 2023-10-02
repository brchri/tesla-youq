package util

import (
	"encoding/xml"
	"log"
	"os"
	"strconv"
	"strings"

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
		Close   []Point `yaml:"close"` // list of points defining a polygon; when vehicle moves from inside this geofence to outside, garage will close
		Open    []Point `yaml:"open"`  // list of points defining a polygon; when vehicle moves from outside this geofence to inside, garage will open
		KMLFile string  `yaml:"kml_file"`
	}

	// kml schema to parse coordinates from kml file for polygon geofences
	KML struct {
		Document struct {
			Placemarks []struct {
				Name    string `xml:"name"`
				Polygon struct {
					OuterBoundary struct {
						LinearRing struct {
							Coordinates string `xml:"coordinates"`
						} `xml:"linearring"`
					} `xml:"outerboundaryis"`
				} `xml:"polygon"`
			} `xml:"placemark"`
		} `xml:"document"`
	}

	Car struct {
		ID                 int         `yaml:"teslamate_car_id"` // mqtt identifier for vehicle
		GarageDoor         *GarageDoor // bidirectional pointer to GarageDoor containing car
		CurrentLocation    Point       // current vehicle location
		LocationUpdate     chan Point  // channel to receive location updates
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
		CircularGeofence  *CircularGeofence  `yaml:"circular_geofence"`
		TeslamateGeofence *TeslamateGeofence `yaml:"teslamate_geofence"`
		PolygonGeofence   *PolygonGeofence   `yaml:"polygon_geofence"`
		MyQSerial         string             `yaml:"myq_serial"`
		Cars              []*Car             `yaml:"cars"` // cars housed within this garage
		OpLock            bool               // controls if garagedoor has been operated recently to prevent flapping
		GeofenceType      string             //indicates whether garage door uses teslamate's geofence or not (checked during runtime)
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
			CacheTokenDir     string `yaml:"cache_token_dir"`
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
	if g.PolygonGeofence != nil && len(g.PolygonGeofence.Open) > 0 && len(g.PolygonGeofence.Close) > 0 {
		return PolygonGeofenceType
	} else if g.CircularGeofence != nil && g.CircularGeofence.Center.IsPointDefined() && g.CircularGeofence.OpenDistance > 0 && g.CircularGeofence.CloseDistance > 0 {
		return CircularGeofenceType
	} else if g.TeslamateGeofence != nil && g.TeslamateGeofence.Close.From != "" &&
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
		// check if kml_file was defined, and if so, load and parse kml and set polygon geofences accordingly
		if g.PolygonGeofence != nil && g.PolygonGeofence.KMLFile != "" {
			loadKMLFile(g.PolygonGeofence)
		}
		g.GeofenceType = g.GetGeofenceType()
		if g.GeofenceType == "" {
			log.Fatalf("error: no supported geofences defined for garage door %v", g)
		}

		// initialize location update channel
		for _, c := range g.Cars {
			c.LocationUpdate = make(chan Point, 2)
		}
	}

	log.Println("Config loaded successfully")
}

// loads kml file and overrides polygon geofence points with parsed data
func loadKMLFile(p *PolygonGeofence) error {
	fileContent, err := os.ReadFile(p.KMLFile)
	lowerKML := strings.ToLower(string(fileContent)) // convert xml to lower to make xml tag parsing case insensitive
	if err != nil {
		log.Printf("Could not read file %s, received error: %e", p.KMLFile, err)
		return err
	}

	var kml KML
	err = xml.Unmarshal([]byte(lowerKML), &kml)
	if err != nil {
		log.Printf("Could not load kml from file %s, received error: %e", p.KMLFile, err)
		return err
	}

	// loop through placemarks to get name and, if relevant, parse the coordinates accordingly
	for _, placemark := range kml.Document.Placemarks {
		var polygonGeoPoints []Point
		// geofences must be named `open` or `close` or they're considered irrelevant
		if placemark.Name != "open" && placemark.Name != "close" {
			continue
		}

		for _, c := range strings.Split(placemark.Polygon.OuterBoundary.LinearRing.Coordinates, "\n") {
			// trim whitespace and continue loop if nothing is left
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}

			// kml coordinate format is longitude,latitude; split comma delim and parse coords
			coords := strings.Split(c, ",")
			lat, err := strconv.ParseFloat(coords[1], 64)
			if err != nil {
				log.Printf("Could not parse lng/lat coordinates from line %s, received error: %e", c, err)
				return err
			}
			lng, err := strconv.ParseFloat(coords[0], 64)
			if err != nil {
				log.Printf("Could not parse lng/lat coordinates from line %s, received error: %e", c, err)
				return err
			}

			polygonGeoPoints = append(polygonGeoPoints, Point{Lat: lat, Lng: lng})
		}

		// set either open or close polygon geo for garage door based on Placemark's Name element
		switch placemark.Name {
		case "open":
			p.Open = polygonGeoPoints
		case "close":
			p.Close = polygonGeoPoints
		}
	}

	return nil
}
