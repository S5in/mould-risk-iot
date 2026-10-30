// Command ingestion subscribes to device telemetry on MQTT and stores valid readings.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/S5in/mould-risk-iot/internal/ingest"
	"github.com/S5in/mould-risk-iot/internal/store"
)

const subscribeTopic = "org/+/device/+/telemetry"

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	password := os.Getenv("INGEST_MQTT_PASSWORD")
	if password == "" {
		return fmt.Errorf("INGEST_MQTT_PASSWORD is not set")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// PG* environment variables (PGHOST, PGUSER, PGPASSWORD, PGDATABASE) configure the connection.
	st, err := store.New(ctx, "")
	if err != nil {
		return err
	}
	defer st.Close()

	h := &ingest.Handler{Store: st, Log: log, Now: time.Now, Timeout: 5 * time.Second}

	broker := env("MQTT_BROKER", "tcp://localhost:1883")
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(env("MQTT_CLIENT_ID", "ingest-service")).
		SetUsername(env("INGEST_MQTT_USER", "ingest")).
		SetPassword(password).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(30 * time.Second).
		SetCleanSession(false)
	// Subscribe in the connect handler so the subscription is restored after every reconnect.
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		tok := c.Subscribe(subscribeTopic, 1, func(_ mqtt.Client, m mqtt.Message) {
			h.Handle(ctx, m.Topic(), m.Payload())
		})
		if tok.Wait() && tok.Error() != nil {
			log.Error("subscribe failed", "err", tok.Error())
			return
		}
		log.Info("subscribed", "topic", subscribeTopic)
	})
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		log.Warn("mqtt connection lost", "err", err)
	})

	client := mqtt.NewClient(opts)
	if tok := client.Connect(); tok.Wait() && tok.Error() != nil {
		return fmt.Errorf("mqtt connect: %w", tok.Error())
	}
	log.Info("ingestion running", "broker", broker)

	<-ctx.Done()
	log.Info("shutting down")
	client.Disconnect(500)
	return nil
}
