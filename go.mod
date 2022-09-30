module github.com/dariopb/gosenseapp

go 1.12

require (
	github.com/dariopb/gosense v0.0.0-00010101000000-000000000000
	github.com/eclipse/paho.mqtt.golang v1.2.0
	github.com/fatih/structs v1.1.0
	github.com/gorilla/websocket v1.4.0
	github.com/labstack/echo/v4 v4.9.0
	github.com/neko-neko/echo-logrus/v2 v2.0.0
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/objx v0.2.0 // indirect
	golang.org/x/tools v0.0.0-20190608022120-eacb66d2a7c3 // indirect
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/dariopb/gosense => ../gosense
