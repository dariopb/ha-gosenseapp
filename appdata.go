package gosenseapp

import (
	"fmt"
	"io/ioutil"
	"math/rand"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func NewAppData(confFilename string) (*ConfigFile, error) {
	c := &ConfigFile{
		Filename: confFilename,
	}

	c, err := c.LoadConfig()
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	if c.AppConfig.DebugLevel == 0 {
		c.AppConfig.DebugLevel = log.InfoLevel
	}

	if c.AppConfig.MQTT.ClientId == "" {
		c.AppConfig.MQTT.ClientId = fmt.Sprintf("ha-gosenseapp-%d", rand.Intn(1000000))
	}
	if c.AppConfig.MQTT.Port == 0 {
		c.AppConfig.MQTT.Port = 1883
	}
	if len(c.AppConfig.MQTT.SensorTopic) == 0 {
		c.AppConfig.MQTT.SensorTopic = "gosense"
	}
	if len(c.AppConfig.MQTT.DiscoveryTopic) == 0 {
		c.AppConfig.MQTT.DiscoveryTopic = "gosense_discovery"
	}

	return c, nil
}

func (c *ConfigFile) Lock() {
	c.mtx.Lock()
}

func (c *ConfigFile) Unlock() {
	c.mtx.Unlock()
}

func (c *ConfigFile) Save(a *AppConfig) error {
	bytes, err := yaml.Marshal(a)
	if err != nil {
		return err
	}

	c.Lock()
	defer c.Unlock()

	return ioutil.WriteFile(c.Filename, bytes, 0644)
}

func (c *ConfigFile) LoadConfig() (*ConfigFile, error) {
	c.Lock()
	defer c.Unlock()

	bytes, err := ioutil.ReadFile(c.Filename)
	if err != nil {
		return nil, err
	}

	var ac AppConfig
	err = yaml.Unmarshal(bytes, &ac)
	if err != nil {
		return nil, err
	}

	if ac.Sensors == nil {
		ac.Sensors = make(map[string]*Sensor)
	}
	c.AppConfig = &ac

	return c, nil
}
