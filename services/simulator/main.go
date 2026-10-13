package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Reading is the JSON message the "device" sends. This is our message contract.
type Reading struct {
	TemperatureC float64   `json:"temperature_c"`
	HumidityRH   float64   `json:"humidity_rh"`
	Timestamp    time.Time `json:"ts"`
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	broker := getenv("MQTT_BROKER", "tcp://localhost:1883")
	user := os.Getenv("MQTT_USER")
	pass := os.Getenv("MQTT_PASS")
	topic := getenv("MQTT_TOPIC", "org/demo-org/device/sim-001/telemetry")

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(user).
		SetUsername(user).
		SetPassword(pass)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("connect failed: %v", token.Error())
	}
	defer client.Disconnect(250)
	log.Printf("connected to %s as %s", broker, user)

	temp, hum := 21.0, 55.0
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	for {
		select {
		case <-ticker.C:
			// random walk: small drift, kept inside plausible limits
			temp += (rand.Float64() - 0.5) * 0.4
			hum += (rand.Float64() - 0.5) * 1.5
			if hum < 20 {
				hum = 20
			}
			if hum > 95 {
				hum = 95
			}

			payload, _ := json.Marshal(Reading{temp, hum, time.Now().UTC()})
			token := client.Publish(topic, 1, false, payload)
			token.Wait()
			if token.Error() != nil {
				log.Printf("publish failed: %v", token.Error())
				continue
			}
			log.Printf("published %s", payload)
		case <-stop:
			log.Println("stopping")
			return
		}
	}
}
