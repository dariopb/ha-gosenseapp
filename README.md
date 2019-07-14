# ha-gosenseapp
This is an application that uses the gosense library (https://github.com/dariopb/gosense) to discover and publish sensor information to HA via the HA MQTT Discovery mechanism (https://www.home-assistant.io/docs/mqtt/discovery/) or simply by configuring the sensor info in the HA config file (https://www.home-assistant.io/docs/mqtt).

# Configuration file

The configuration file for the application. At least an empty file needs to be provided and it will be populated with default values.

* **debuglevel**          The debug level for tracing info/debug [defaults to info if it is not set]

*MQTT section*

* **clientid**          The MQTT clientID to use [defaults to "ha-gosenseapp-xxxxx" if it is not set]
* **hostname**          The MQTT server to connect to. Empty disables MQTT [defaults to ""]
* **user**              The MQTT username to use [defaults to "" is not set]
* **password**          The MQTT username to use [defaults to "" is not set]
* **port**              The MQTT port to use [defaults to 1883]
* **discoveryTopic**    The MQTT-DISC for HA topic to use [defaults to "gosense_discovery" if it is not set]
* **sensorTopic**    The MQTT clientID to use [defaults to "gosense" if it is not set]

Sample configuration file

```yaml
mqtt:
  clientid: goapp-001
  hostname: localhost
  user: ""
  password: ""
  port: 1883
  sensorTopic: gosense
  discoveryTopic: gosense_discovery

sensors:    <== sensor data is discovered and updated from the dongle
  0444001:
    metadata:
      name: 0444001
      mac: 0444001
      sensorType: 1
      present: true
    properties:
      battery: "94"
      signal: "40"
      state: "0"
      timeLastAlarm: "2019-07-06T00:20:09-07:00"
```

# Run it!

Linux 

````
# Create a config file as above and then mount that file to /app.yaml :
sudo docker run -it --rm --net host -v /home/xxxx/gosenseapp/app.yaml:/app.yaml --privileged dariob/gosenseapp:latest
````

Raspberry PI 

````
# Create a config file as above and then mount that file:
sudo docker run -it --rm --net host -v /home/pi/gosenseapp/app.yaml:/app.yaml --privileged dariob/gosenseapp-pi:latest
````


# MQTT Structure
This is the MQTT structure for the default MQTT settings:

````yaml
├── gosense-discovery
│   └── binary_sensor
│       └── 0444001
│           └── config
│               └── JSON PAYLOAD
                      name: "PPPPP"
                      payload_on: "1"
                      payload_off: "0"
                      state_topic: "gosense/0444001"
                      value_template: "{{ value_json.properties.state }}"
                      json_attributes_topic: "gosense/0444001"
                      json_attributes_template: "{{ value_json.properties | tojson }}"
...
├── gosense
│   └── 0444002
│           └── JSON PAYLOAD
                    metadata:
                      name: 0444002
                      mac: 0444002
                      sensorType: 1
                      present: true
                    properties:
                      battery: '94'
                      signal: '40'
                      state: '1'
                      timeLastAlarm: '2019-07-06T00:20:09-07:00'

````

# REST API
The application exposes a simple REST API to manipule the sense dongle and get "configuration ready" output to include in the HA conf file.

**Routes exposed**


* **/help**    Gets the list of exposed routes

* **./hasensors**    Enumerates the sensors attached to the dongle in HA conf format
````yaml
binarySensor:
- platform: mqttv
  name: 0444001
  unique_id: 0444001
  payload_on: "1"
  payload_off: "0"
  state_topic: gosense/0444001
  value_template: '{{ value_json.properties.state }}'
  json_attributes_topic: gosense/0444001
  json_attributes_template: '{{ value_json.properties | tojson }}'
- platform: mqtt
  name: 0444002
  unique_id: 0444002
  payload_on: "1"
  payload_off: "0"
  state_topic: gosense/0444002
  value_template: '{{ value_json.properties.state }}'
  json_attributes_topic: gosense/0444002
  json_attributes_template: '{{ value_json.properties | tojson }}'
````

* **/sensors**    Enumerates the sensors attached to the dongle
````json
[{"metadata":{"name":"","mac":"","sensorType":0},"properties":{}},{"metadata":{"name":"0444001","mac":"0444001","sensorType":1},"properties":{"battery":"96","signal":"41","state":"0","timeLastAlarm":"2019-07-13T15:39:52-07:00"}},{"metadata":{"name":"0444002","mac":"0444002","sensorType":2},"properties":{"battery":"0","signal":"1","state":"5","timeLastAlarm":"2019-07-13T16:03:08-07:00"}}]
````
* **/sensors/:mac**    Get a sensor by its MAC address
* **/sensors/scan**    Sets the dongle in attach mode for 30 seconds. You can pair a new device in that time. The info about the new device is the response.
````json
{"metadata":{"name":"0444002","mac":"0444002","sensorType":0},"properties":{}}
````
* **/sensors/:mac/remove**    Removes (unpairs) a sensor by MAC address



# Docker

Create docker image:

````
cd cmd
env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o gosenseapp
cd ..
sudo docker build -t dariob/gosenseapp:latest -f docker/Dockerfile .
````

Run on Raspberry PI

Please notice the --privileged flag needs to be passed in order to be able to access the host device.
````
root@rasp1:/home/pi/gosenseapp# docker run -it --rm --net host -v /home/pi/gosenseapp/app.yaml:/app.yaml --privileged dariob/gosenseapp-pi:latest
Unable to find image 'dariob/gosenseapp-pi:latest' locally
latest: Pulling from dariob/gosenseapp-pi
13ee9c2f1d69: Pull complete 
Digest: sha256:d2256eae3f6056dfd771c076b786b13de9a0c03375fd5499a64193f8f496747e
Status: Downloaded newer image for dariob/gosenseapp-pi:latest
ha-gosenseapp starting...
INFO[0000] Looking for sense dongle in /dev/hidraw*     
INFO[0000] Found sense dongle in [/dev/hidraw2]         
INFO[0000] NewWyzeSense: trying to open device: [/dev/hidraw2] 
INFO[0000] LOG: time=14 Jul 19 01:04 +0000, data=�     
INFO[0000] Starting echo REST API on port 8080          
INFO[0000] Starting HA MQTT publisher to [tcp://192.168.1.108:1883] 
INFO[0000]    MQTT trying to connect to: [tcp://192.168.1.108:1883] 
WARN[0000]    MQTT sucessfully connected to: [tcp://192.168.1.108:1883] 
INFO[0005] LOG: time=14 Jul 19 01:05 +0000, data=�77793E5A 
INFO[0005] ALARM: time=14 Jul 19 01:05 +0000, mac: 77793E5A, type: 1, battery: 97, signal: 91, state: 1, data=61000101000a5b 
INFO[0005] ALARM: 77793E5A, state: 1                    
INFO[0008] LOG: time=14 Jul 19 01:05 +0000, data=�77793E5A
                                                            
INFO[0008] ALARM: time=14 Jul 19 01:05 +0000, mac: 77793E5A, type: 1, battery: 97, signal: 82, state: 0, data=61000100000b52 
INFO[0008] ALARM: 77793E5A, state: 0                    

````

# Raspberry PI 

````
cd cmd
env GOARCH=arm GOARM=5 GOOS=linux go build -o gosenseapp
#scp gosenseapp rasp:~/gosenseapp

cd ..
sudo docker build -t dariob/gosenseapp-pi:latest -f docker/Dockerfile .

````