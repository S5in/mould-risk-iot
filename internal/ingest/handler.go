// Package ingest turns MQTT messages into stored readings.
package ingest

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/S5in/mould-risk-iot/internal/store"
	"github.com/S5in/mould-risk-iot/internal/telemetry"
)

// Inserter is what the handler needs from the database layer. Tests use a fake.
type Inserter interface {
	InsertReading(ctx context.Context, orgSlug, deviceKey string, at time.Time, tempC, humRH float64) error
}

// Outcome says what happened to a message.
type Outcome string

const (
	Stored    Outcome = "stored"
	Rejected  Outcome = "rejected"
	Duplicate Outcome = "duplicate"
	Failed    Outcome = "failed"
)

// Handler validates and stores messages.
type Handler struct {
	Store   Inserter
	Log     *slog.Logger
	Now     func() time.Time
	Timeout time.Duration
}

// Handle processes one message. It never panics on bad input and never logs the payload body.
func (h *Handler) Handle(ctx context.Context, topic string, payload []byte) Outcome {
	t, err := telemetry.ParseTopic(topic)
	if err != nil {
		h.Log.Warn("rejected", "reason", "bad topic", "err", err)
		return Rejected
	}
	r, err := telemetry.ParseReading(payload, h.Now())
	if err != nil {
		h.Log.Warn("rejected", "org", t.OrgSlug, "device", t.DeviceKey, "err", err)
		return Rejected
	}
	ctx, cancel := context.WithTimeout(ctx, h.Timeout)
	defer cancel()
	err = h.Store.InsertReading(ctx, t.OrgSlug, t.DeviceKey, r.Timestamp, r.TemperatureC, r.HumidityRH)
	switch {
	case err == nil:
		h.Log.Info("stored", "org", t.OrgSlug, "device", t.DeviceKey, "temperature_c", r.TemperatureC, "humidity_rh", r.HumidityRH)
		return Stored
	case errors.Is(err, store.ErrUnknownDevice):
		h.Log.Warn("rejected", "org", t.OrgSlug, "device", t.DeviceKey, "err", err)
		return Rejected
	case errors.Is(err, store.ErrDuplicate):
		h.Log.Info("duplicate", "org", t.OrgSlug, "device", t.DeviceKey)
		return Duplicate
	default:
		h.Log.Error("database error", "org", t.OrgSlug, "device", t.DeviceKey, "err", err)
		return Failed
	}
}
