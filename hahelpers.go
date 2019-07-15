package gosenseapp

import (
	"fmt"

	"github.com/fatih/structs"
)

type haconfig struct {
	BinarySensors []binarySensor `yaml:"binarySensor" json:"binarySensor"`
}

type binarySensor struct {
	Platform               string `yaml:"platform" json:"platform"`
	Name                   string `yaml:"name" json:"name"`
	UniqueID               string `yaml:"unique_id" json:"unique_id"`
	PayloadOn              string `yaml:"payload_on" json:"payload_on"`
	PayloadOff             string `yaml:"payload_off" json:"payload_off"`
	StateTopic             string `yaml:"state_topic" json:"state_topic"`
	ValueTemplate          string `yaml:"value_template" json:"value_template"`
	JSONAttributesTopic    string `yaml:"json_attributes_topic" json:"json_attributes_topic"`
	JSONAttributesTemplate string `yaml:"json_attributes_template" json:"json_attributes_template"`

	/*	binary_sensor:
		- platform: mqtt
		  name: "PPPPP"
		  payload_on: "1"
		  payload_off: "0"
		  state_topic: "gosense/mac1234"
		  value_template: "{{ value_json.properties.state }}"
		  json_attributes_topic: "gosense/mac1234"
		  json_attributes_template: "{{ value_json.properties | tojson }}"
	*/
}

func binarySensorConfigFromSensor(topic string, v *Sensor) binarySensor {
	sensorTopic := fmt.Sprintf("%s/%s", topic, v.Metadata.MAC)

	bs := binarySensor{
		Platform:               "mqtt",
		Name:                   v.Metadata.Name,
		UniqueID:               v.Metadata.MAC,
		PayloadOn:              "1",
		PayloadOff:             "0",
		StateTopic:             sensorTopic,
		ValueTemplate:          "{{ value_json.properties.state }}",
		JSONAttributesTopic:    sensorTopic,
		JSONAttributesTemplate: "{{ value_json.properties | tojson }}",
	}

	return bs
}

func getPropertyMap(obj interface{}) *map[string]interface{} {
	m := structs.Map(obj)

	return &m
}
