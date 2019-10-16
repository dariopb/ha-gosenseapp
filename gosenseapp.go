package gosenseapp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	gosense "github.com/dariopb/gosense"
	log "github.com/sirupsen/logrus"
)

type DeviceMetadata struct {
	Name       string             `yaml:"name" json:"name"`
	MAC        string             `yaml:"mac" json:"mac"`
	SensorType gosense.SensorType `yaml:"sensorType" json:"sensorType"`
	Present    bool               `yaml:"present" json:"present"`
}

type DeviceProperties struct {
}

type Sensor struct {
	Metadata   DeviceMetadata    `yaml:"metadata" json:"metadata"`
	Properties map[string]string `yaml:"properties" json:"properties"`
}

type MQTT struct {
	ClientId       string `yaml:"clientid" json:"clientid"`
	Hostname       string `yaml:"hostname" json:"hostname"`
	User           string `yaml:"user" json:"user"`
	Password       string `yaml:"password" json:"password"`
	Port           uint   `yaml:"port" json:"port"`
	SensorTopic    string `yaml:"sensorTopic" json:"sensorTopic"`
	DiscoveryTopic string `yaml:"discoveryTopic" json:"discoveryTopic"`
}

type ConfigFile struct {
	Filename  string `yaml:"ignore" json:"ignore"`
	AppConfig *AppConfig

	mtx sync.Mutex
}

type AppConfig struct {
	DebugLevel log.Level          `yaml:"debuglevel" json:"debuglevel"`
	MQTT       MQTT               `yaml:"mqtt" json:"mqtt"`
	Sensors    map[string]*Sensor `yaml:"sensors" json:"sensors"`
}

var SenseData *ConfigFile
var SenseDev *gosense.WyzeSense

// FindSenseDevice tries to find the first sense dongle plugged in.
func FindSenseDevice() (string, error) {
	devname := ""
	root := "/sys/class/hidraw"
	names, err := filepath.Glob(path.Join(root, "*"))

	log.Info("Looking for sense dongle in /dev/hidraw*")
loop:
	for _, dirname := range names {
		fname := path.Join(dirname, "device/uevent")
		file, err := os.Open(fname)
		if err != nil {
			continue
		}

		defer file.Close()

		reader := bufio.NewReader(file)

		for {
			line, _, err := reader.ReadLine()

			if err == io.EOF {
				break
			}
			if strings.HasPrefix(string(line), "HID_ID=") &&
				strings.Contains(string(line), "00001A86:0000E024") {
				_, devname = path.Split(dirname)
				devname = path.Join("/dev", devname)
				break loop
			}
		}
	}
	if err != nil {
		return "", err
	}

	if len(devname) > 0 {
		log.Infof("Found sense dongle in [%s]", devname)
	} else {
		err = fmt.Errorf("can't find sense device")
	}

	return devname, err
}

func Run() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.SetOutput(os.Stdout)

	var err error
	confFilename, ok := os.LookupEnv("CONFIG_FILE")
	if !ok {
		confFilename = "app.yaml"
	}
	SenseData, err = NewAppData(confFilename)
	if err != nil {
		os.Exit(3)
	}

	if SenseData.AppConfig.DebugLevel != 0 {
		log.SetLevel(SenseData.AppConfig.DebugLevel)
	}

	devicename, err := FindSenseDevice() // "/dev/hidraw2"
	if err != nil {
		log.Error(err.Error())
		os.Exit(3)
	}

	alarmch := make(chan (gosense.Alarm), 10)
	sensorch := make(chan (gosense.SenseSensor), 10)

	s, err := gosense.NewWyzeSense(devicename, alarmch, sensorch)
	if err != nil {
		log.Error(err.Error())
		os.Exit(3)
	}
	SenseDev = s

	err = SenseDev.Start()
	if err != nil {
		log.Error(err.Error())
		os.Exit(3)
	}

	//s.ScanSensor()
	//s.DeleteSensor("7779768D")
	//s.VerifySensor("7779768D")

	// Start the REST api for easy qurying/managing
	api, err := NewRestApi(8080)
	if err != nil {
		log.Error(err.Error())
		os.Exit(3)
	}

	var mqtt *MQTTPublisher

	if len(SenseData.AppConfig.MQTT.Hostname) > 0 {
		// Start the MQTT publisher
		mqtt = NewMQTTPublisher(SenseData.AppConfig.MQTT)
		//if err != nil {
		//	log.Error(err.Error())
		//	os.Exit(3)
		//}
	} else {
		log.Info("  => skipping HA MQTT publisher since no server was provided")
	}

	// Merge the data I have with the data collected
	sensors, err := SenseDev.GetSensorList()
	if err != nil {
		log.Error(err.Error())
		os.Exit(3)
	}

	SenseData.Lock()
	for _, mac := range sensors {
		sensor, ok := SenseData.AppConfig.Sensors[mac]

		if !ok {
			sensor = &Sensor{
				Metadata: DeviceMetadata{
					Name:    mac,
					MAC:     mac,
					Present: true,
				},
				Properties: make(map[string]string),
			}

			SenseData.AppConfig.Sensors[mac] = sensor
		}

		if mqtt != nil {
			mqtt.sensorch <- *sensor
		}
	}
	SenseData.Unlock()
	SenseData.Save(SenseData.AppConfig)

loop:
	for {
		select {
		case a := <-alarmch:
			log.Infof("ALARM: %s, state: %d",
				a.MAC, a.State)
			// Only Alarms with flag 0xA2 (162) signal an open/close/motion alarm
			if int(a.SensorFlags) != 162 {
				log.Infof("ALARM: Ignoring Alarm with unknown flag: %x",
					  a.SensorFlags)
				break
			}
			
			SenseData.Lock()
			sensor, ok := SenseData.AppConfig.Sensors[a.MAC]
			if ok {
				sensor.Metadata.SensorType = a.SensorType
				sensor.Properties["state"] = strconv.Itoa(int(a.State))
				sensor.Properties["battery"] = strconv.Itoa(int(a.Battery))
				sensor.Properties["signal"] = strconv.Itoa(int(a.SignalStrength))
				sensor.Properties["timeLastAlarm"] = a.Timestamp.Format(time.RFC3339)
			}
			SenseData.Unlock()
			SenseData.Save(SenseData.AppConfig)

			if mqtt != nil {
				mqtt.alarmch <- *sensor
			}
			break
		case sensorAct := <-sensorch:
			log.Infof("SENSOR: %s, state: %d [present: %t]",
				sensorAct.MAC, sensorAct.SensorType, sensorAct.Present)

			SenseData.Lock()
			var sensor = &Sensor{
				Metadata: DeviceMetadata{
					Name:    sensorAct.MAC,
					MAC:     sensorAct.MAC,
					Present: sensorAct.Present,
				},
				Properties: make(map[string]string),
			}

			if sensorAct.Present {
				var sensor *Sensor
				sensor, ok := SenseData.AppConfig.Sensors[sensorAct.MAC]
				if !ok {
					sensor = &Sensor{
						Metadata: DeviceMetadata{
							Name:    sensorAct.MAC,
							MAC:     sensorAct.MAC,
							Present: sensorAct.Present,
						},
						Properties: make(map[string]string),
					}
				}
				sensor.Metadata.Present = sensorAct.Present
				SenseData.AppConfig.Sensors[sensorAct.MAC] = sensor
			} else {
				delete(SenseData.AppConfig.Sensors, sensorAct.MAC)
			}

			if mqtt != nil {
				mqtt.sensorch <- *sensor
			}

			SenseData.Unlock()
			SenseData.Save(SenseData.AppConfig)
			break
		case <-c:
			break loop
		}
	}

	mqtt.Close()
	api.Close()
}
