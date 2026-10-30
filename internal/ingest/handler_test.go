package ingest

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/S5in/mould-risk-iot/internal/store"
)

type fakeStore struct {
	err   error
	calls int
	org   string
	dev   string
}

func (f *fakeStore) InsertReading(_ context.Context, org, dev string, _ time.Time, _, _ float64) error {
	f.calls++
	f.org, f.dev = org, dev
	return f.err
}

var fixedNow = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

const goodTopic = "org/demo-org/device/sim-001/telemetry"
const goodPayload = `{"temperature_c":21.5,"humidity_rh":55.2,"ts":"2026-10-30T11:59:50Z"}`

func newHandler(f *fakeStore) *Handler {
	return &Handler{
		Store:   f,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:     func() time.Time { return fixedNow },
		Timeout: time.Second,
	}
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name      string
		topic     string
		payload   string
		storeErr  error
		want      Outcome
		wantCalls int
	}{
		{"valid reading is stored", goodTopic, goodPayload, nil, Stored, 1},
		{"bad topic is rejected before the database", "org/demo-org/telemetry", goodPayload, nil, Rejected, 0},
		{"invalid JSON is rejected", goodTopic, `{not json`, nil, Rejected, 0},
		{"humidity over 100 is rejected", goodTopic, `{"temperature_c":21,"humidity_rh":150,"ts":"2026-10-30T11:59:50Z"}`, nil, Rejected, 0},
		{"missing field is rejected", goodTopic, `{"temperature_c":21,"ts":"2026-10-30T11:59:50Z"}`, nil, Rejected, 0},
		{"future timestamp is rejected", goodTopic, `{"temperature_c":21,"humidity_rh":50,"ts":"2026-10-30T13:00:00Z"}`, nil, Rejected, 0},
		{"unknown device is rejected", goodTopic, goodPayload, store.ErrUnknownDevice, Rejected, 1},
		{"duplicate is reported", goodTopic, goodPayload, store.ErrDuplicate, Duplicate, 1},
		{"database failure is reported", goodTopic, goodPayload, errors.New("connection refused"), Failed, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeStore{err: tc.storeErr}
			got := newHandler(f).Handle(context.Background(), tc.topic, []byte(tc.payload))
			if got != tc.want {
				t.Errorf("outcome = %q, want %q", got, tc.want)
			}
			if f.calls != tc.wantCalls {
				t.Errorf("store calls = %d, want %d", f.calls, tc.wantCalls)
			}
		})
	}
}

func TestHandlePassesTopicIdentityToStore(t *testing.T) {
	f := &fakeStore{}
	newHandler(f).Handle(context.Background(), goodTopic, []byte(goodPayload))
	if f.org != "demo-org" || f.dev != "sim-001" {
		t.Errorf("store got org=%q device=%q", f.org, f.dev)
	}
}
