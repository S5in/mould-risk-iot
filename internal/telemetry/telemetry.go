package telemetry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	maxPayloadBytes = 1024
	maxFutureSkew   = 5 * time.Minute
)

// Reading is a validated sensor reading.
type Reading struct {
	TemperatureC float64
	HumidityRH   float64
	Timestamp    time.Time
}

// wireReading is the raw JSON shape. Pointers let us detect missing fields.
type wireReading struct {
	TemperatureC *float64   `json:"temperature_c"`
	HumidityRH   *float64   `json:"humidity_rh"`
	Timestamp    *time.Time `json:"ts"`
}

// Topic identifies who a message claims to be from.
type Topic struct {
	OrgSlug   string
	DeviceKey string
}

// ParseTopic accepts only org/{org}/device/{device}/telemetry.
func ParseTopic(topic string) (Topic, error) {
	parts := strings.Split(topic, "/")
	if len(parts) != 5 || parts[0] != "org" || parts[2] != "device" || parts[4] != "telemetry" {
		return Topic{}, fmt.Errorf("unexpected topic %q", topic)
	}
	if parts[1] == "" || parts[3] == "" {
		return Topic{}, errors.New("empty org or device in topic")
	}
	return Topic{OrgSlug: parts[1], DeviceKey: parts[3]}, nil
}

// ParseReading strictly decodes and validates a payload. now is passed in so tests are deterministic.
func ParseReading(payload []byte, now time.Time) (Reading, error) {
	if len(payload) > maxPayloadBytes {
		return Reading{}, errors.New("payload too large")
	}
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.DisallowUnknownFields()
	var w wireReading
	if err := dec.Decode(&w); err != nil {
		return Reading{}, fmt.Errorf("invalid JSON: %w", err)
	}
	if w.TemperatureC == nil || w.HumidityRH == nil || w.Timestamp == nil {
		return Reading{}, errors.New("missing required field")
	}
	if *w.TemperatureC < -40 || *w.TemperatureC > 85 {
		return Reading{}, fmt.Errorf("temperature %.1f out of range", *w.TemperatureC)
	}
	if *w.HumidityRH < 0 || *w.HumidityRH > 100 {
		return Reading{}, fmt.Errorf("humidity %.1f out of range", *w.HumidityRH)
	}
	if w.Timestamp.After(now.Add(maxFutureSkew)) {
		return Reading{}, errors.New("timestamp is in the future")
	}
	return Reading{*w.TemperatureC, *w.HumidityRH, *w.Timestamp}, nil
}
