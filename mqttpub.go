package gosenseapp

import (
	"encoding/json"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

type MQTTPublisher struct {
	serverURI          string
	topicRoot          string
	discoveryTopicRoot string
	clientOptions      *mqtt.ClientOptions
	client             mqtt.Client

	alarmch  chan Sensor
	sensorch chan Sensor
	sc       chan bool
}

func NewMQTTPublisher(conf MQTT) *MQTTPublisher {
	p := MQTTPublisher{
		topicRoot:          SenseData.AppConfig.MQTT.SensorTopic,
		discoveryTopicRoot: SenseData.AppConfig.MQTT.DiscoveryTopic,
		alarmch:            make(chan (Sensor), 1),
		sensorch:           make(chan (Sensor), 1),
		sc:                 make(chan (bool)),
	}
	if conf.Port == 0 {
		conf.Port = 1883
	}

	p.serverURI = fmt.Sprintf("tcp://%s:%d", conf.Hostname, conf.Port)
	log.Infof("Starting HA MQTT publisher to [%s]", p.serverURI)

	p.clientOptions = mqtt.NewClientOptions()
	p.clientOptions.AddBroker(p.serverURI)

	if SenseData.AppConfig.MQTT.User != "" {
		log.Infof("    -> using username: [%s]", SenseData.AppConfig.MQTT.User)
		p.clientOptions.SetUsername(SenseData.AppConfig.MQTT.User)
		p.clientOptions.SetPassword(SenseData.AppConfig.MQTT.Password)
	}
	p.clientOptions.SetClientID(SenseData.AppConfig.MQTT.ClientId)
	p.clientOptions.SetCleanSession(false)
	p.clientOptions.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Infof("   MQTT publish: [%s] -> [%s]", msg.Topic(), string(msg.Payload()))
	})

	p.clientOptions.SetKeepAlive(5 * time.Second)
	p.clientOptions.SetConnectTimeout(2 * time.Second)
	p.clientOptions.SetMaxReconnectInterval(2 * time.Second)
	p.clientOptions.SetAutoReconnect(true)

	p.clientOptions.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Warningf("   MQTT connection lost: [%s] -> [%s]", p.serverURI, err.Error())
	})

	p.clientOptions.SetOnConnectHandler(func(c mqtt.Client) {
		log.Warningf("   MQTT sucessfully connected to: [%s]", p.serverURI)
	})

	p.client = mqtt.NewClient(p.clientOptions)

	go func() {
		p.loop()
		//if err != nil {
		//	log.Errorf("Start echo failed with [%s]", err.Error())
		//	panic(err.Error())
		//}
	}()

	return &p
}

func (p MQTTPublisher) Close() {
	log.Info("Stopping HA MQTT publisher")
	p.sc <- true
}

func (p MQTTPublisher) ensureConnected() {
	for {
		token := p.client.Connect()
		token.Wait()

		if token.Error() == nil {
			break
		}

		log.Warningf("   MQTT connect to: [%s], error: %v", p.serverURI, token.Error())
		time.Sleep(2 * time.Second)
	}
}

func (p MQTTPublisher) loop() {
	log.Infof("   MQTT trying to connect to: [%s]", p.serverURI)

	// Need to connect at least once for reconnection to take place
	p.ensureConnected()

loop:
	for {
		select {
		case sensorData := <-p.sensorch:
			discoveryTopic := fmt.Sprintf("%s/binary_sensor/%s/config", p.discoveryTopicRoot, sensorData.Metadata.MAC)
			bs := binarySensorConfigFromSensor(p.topicRoot, &sensorData)

			json, err := json.Marshal(bs)
			if err != nil {
				continue
			}
			if p.client.IsConnectionOpen() {
				// Remove the previous definition. Is this the right way to update it??
				token := p.client.Publish(discoveryTopic, 1, true, "")
				token.WaitTimeout(5 * time.Second)
				if token.Error() != nil {
					log.Infof("   MQTT lost connection to broker: [%s]", token.Error())
				}

				if sensorData.Metadata.Present {
					token = p.client.Publish(discoveryTopic, 1, true, string(json))
					token.WaitTimeout(5 * time.Second)
					if token.Error() != nil {
						log.Infof("   MQTT lost connection to broker: [%s]", token.Error())
					}
				} else {

				}
			} else {
				log.Infof("   MQTT disconnected from broker: [%s]", p.serverURI)
			}

		case sensorEvt := <-p.alarmch:
			mqttTopic := fmt.Sprintf("%s/%s", p.topicRoot, sensorEvt.Metadata.MAC)
			json, err := json.Marshal(sensorEvt)
			if err != nil {
				continue
			}

			if p.client.IsConnectionOpen() {
				token := p.client.Publish(mqttTopic, 1, true, string(json))
				token.WaitTimeout(5 * time.Second)
				if token.Error() != nil {
					log.Infof("   MQTT lost connection to broker: [%s]", token.Error())
				}
			} else {
				log.Infof("   MQTT disconnected from broker: [%s]", p.serverURI)
			}
		case <-p.sc:
			break loop
		}
	}
}
