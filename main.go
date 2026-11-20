package main

import (
	"flag"
	"log"

	"github.com/lomik/noolite2mqtt/pkg/hub"
	"github.com/lomik/noolite2mqtt/pkg/mtrf"
)

func onMessage(topic string, payload []byte) {
	log.Println("on message", topic, string(payload))
}

func main() {
	port := flag.String("port", "/dev/ttyAMA0", "Serial port")
	broker := flag.String("server", "127.0.0.1:1883", "MQTT broker")
	topic := flag.String("topic", "noolite2mqtt", "MQTT root topic")
	mqttClientID := flag.String("client", "noolite2mqtt", "MQTT client ID")
	mqttUser := flag.String("user", "", "MQTT user")
	mqttPassword := flag.String("password", "", "MQTT password")
	watchdog := flag.Duration("watchdog", 0, "Exit if MTRF module does not respond for this long (module is probed every 1/3 of it). 0 - disabled")

	flag.Parse()

	device := mtrf.Connect(*port)

	h, err := hub.New(device, hub.Options{
		Broker:   *broker,
		Topic:    *topic,
		ClientID: *mqttClientID,
		User:     *mqttUser,
		Password: *mqttPassword,
	})

	if err != nil {
		log.Fatal(err)
	}

	if *watchdog > 0 {
		go h.Watchdog(*watchdog, func() {
			log.Fatalf("watchdog: no response from MTRF module for %s", *watchdog)
		})
	}

	h.Loop()
}
