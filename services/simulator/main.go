// Command simulator publishes fake sensor readings, standing in for the ESP32.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type reading struct {
	TemperatureC float64 `json:"temperature_c"`
	HumidityRH   float64 `json:"humidity_rh"`
	Timestamp    string  `json:"ts"`
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	interval := flag.Duration("interval", 5*time.Second, "time between readings")
	count := flag.Int("count", 0, "stop after this many readings (0 = run until Ctrl+C)")
	bad := flag.Bool("bad", false, "publish an invalid reading (humidity 150) to show it is rejected")
	flag.Parse()

	org := env("SIM_ORG", "demo-org")
	device := env("SIM_DEVICE", "sim-001")
	password := os.Getenv("SIM_MQTT_PASSWORD")
	if password == "" {
		log.Fatal("SIM_MQTT_PASSWORD is not set")
	}
	topic := fmt.Sprintf("org/%s/device/%s/telemetry", org, device)

	opts := mqtt.NewClientOptions().
		AddBroker(env("MQTT_BROKER", "tcp://localhost:1883")).
		SetClientID(device).
		SetUsername(device).
		SetPassword(password).
		SetAutoReconnect(true)
	client := mqtt.NewClient(opts)
	if tok := client.Connect(); tok.Wait() && tok.Error() != nil {
		log.Fatalf("connect: %v", tok.Error())
	}
	defer client.Disconnect(250)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	start := time.Now()
	for n := 1; ; n++ {
		minutes := time.Since(start).Minutes()
		r := reading{
			TemperatureC: round1(21 + 1.5*math.Sin(minutes/5) + rand.Float64()*0.2),
			HumidityRH:   round1(55 + 8*math.Sin(minutes/7) + rand.Float64()*0.5),
			Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		}
		if *bad {
			r.HumidityRH = 150
		}
		body, err := json.Marshal(r)
		if err != nil {
			log.Fatalf("marshal: %v", err)
		}
		tok := client.Publish(topic, 1, false, body)
		tok.Wait()
		if tok.Error() != nil {
			log.Printf("publish failed: %v", tok.Error())
		} else {
			log.Printf("published to %s: %s", topic, body)
		}
		if *count > 0 && n >= *count {
			return
		}
		select {
		case <-ticker.C:
		case <-interrupt:
			return
		}
	}
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
