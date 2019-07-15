


binary_sensor
  - platform: mqtt
    name: "PPPPP"
    unique_id: mac1234
    device_class: motion
    payload_on: "1"
    payload_off: "0"
    state_topic: "gosense/sensor/node1/mac1234"
    value_template: "{{ value_json.properties.state }}"
    json_attributes_topic: "gosense/sensor/node1/mac1234"
    json_attributes_template: "{{ value_json.properties | tojson }}"

# Publish for MQTT Discovery
mosquitto_pub -h localhost -p 1883 -q 1 -t 'gosense_discovery/binary_sensor/node1/mac1234/config' -r -m ''
mosquitto_pub -h localhost -p 1883 -q 1 -t 'gosense_discovery/binary_sensor/node1/mac1234/config' -r -m '{"name":"mac1234", "unique_id":"mac1234", "device_class":"door", "payload_on":"1", "payload_off": "0", "state_topic":"gosense/mac1234", "value_template":"{{ value_json.properties.state}}", "json_attributes_topic":"gosense/mac1234", "json_attributes_template": "{{ value_json.properties | tojson }}"}'




    templates:
      rgb_color: "if (state === '1') return [251, 210, 41]; else return [54, 95, 140];"


templates: rgb_color: "if (state === 'on') return [251, 210, 41]; else return [54, 95, 140];"


# Publish state
mosquitto_pub -h localhost -p 1883 -q 1 -t 'gosense/mac1234' -m '{"metadata":{"name":"7779768D","mac":"7779768D","sensorType":1},"properties":{"battery":"94","signal":"40","state":"1","timeLastAlarm":"2019-07-06T00:20:09-07:00"}}' 
