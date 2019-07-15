package gosenseapp

import (
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/labstack/echo/v4"
	middleware "github.com/neko-neko/echo-logrus/v2"
	"github.com/neko-neko/echo-logrus/v2/log"
	"gopkg.in/yaml.v2"
)

type RestAPI struct {
	echo *echo.Echo
}

func NewRestApi(port int) (RestAPI, error) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Routes
	e.GET("/help", help)
	e.GET("/sensors", getSensors)
	e.GET("/sensors/:mac", getSensors)
	e.GET("/sensors/scan", discoverSensor)
	e.GET("/sensors/:mac/remove", removeSensor)
	e.GET("/hasensors", getHASensors)

	//e.Logger = echolog.NewLogger(logrus.StandardLogger(), "")
	e.Logger = log.Logger()
	e.Use(middleware.Logger())

	log.Infof("Starting echo REST API on port %d", port)

	go func() {
		err := e.Start(fmt.Sprintf(":%d", port))
		if err != nil {
			log.Errorf("Start echo failed with [%s]", err.Error())
			panic(err.Error())
		}
	}()

	api := RestAPI{
		echo: e,
	}

	return api, nil
}

func (r RestAPI) Close() {
	if r.echo != nil {
		r.echo.Close()
		r.echo = nil
	}
}

func help(c echo.Context) error {
	hostname, _ := os.Hostname()
	banner := fmt.Sprintf("SenseRestAPI alive on [%s] (%s on %s/%s). Available routes: ",
		hostname, runtime.Version(), runtime.GOOS, runtime.GOARCH)

	routes := c.Echo().Routes()
	for _, route := range routes {
		banner = banner + "<li>" + route.Path + "</li> "
	}

	return c.HTML(http.StatusOK, banner)
}

func getSensors(c echo.Context) error {
	sensors := make([]Sensor, 0)
	mac := c.Param("mac")
	var ret interface{}

	SenseData.Lock()
	defer SenseData.Unlock()

	for _, v := range SenseData.AppConfig.Sensors {
		if len(mac) == 0 {
			sensors = append(sensors, *v)
			ret = sensors
		} else {
			if mac == v.Metadata.MAC {
				ret = v
				break
			}
		}
	}

	if ret == nil {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Sensor [%s] not found", mac))
	}

	return c.JSON(http.StatusOK, ret)
}

// Get the sensors in HA ready format
//
// binary_sensor:
//  - platform: mqtt
//    name: "PPPPP"
//    payload_on: "1"
//    payload_off: "0"
//    state_topic: "gosense/mac1234"
//    value_template: "{{ value_json.properties.state }}"
//    json_attributes_topic: "gosense/mac1234"
//    json_attributes_template: "{{ value_json.properties | tojson }}"

func getHASensors(c echo.Context) error {
	SenseData.Lock()
	defer SenseData.Unlock()

	ha := haconfig{}
	bsa := make([]binarySensor, 0)

	topic := SenseData.AppConfig.MQTT.SensorTopic

	for _, v := range SenseData.AppConfig.Sensors {
		bs := binarySensorConfigFromSensor(topic, v)
		bsa = append(bsa, bs)
	}

	ha.BinarySensors = bsa
	data, err := yaml.Marshal(ha)
	if err != nil {
		return err
	}

	return c.Blob(http.StatusOK, "text/yaml", data)
}

func discoverSensor(c echo.Context) error {
	var ret interface{}
	mac, err := SenseDev.ScanSensor()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	SenseData.Lock()
	defer SenseData.Unlock()

	if s, ok := SenseData.AppConfig.Sensors[mac]; ok {
		ret = s
	}
	return c.JSON(http.StatusOK, ret)
}

func removeSensor(c echo.Context) error {
	mac := c.Param("mac")
	if len(mac) == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "MAC address was not provided")
	}

	err := SenseDev.DeleteSensor(mac)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, "")
}
