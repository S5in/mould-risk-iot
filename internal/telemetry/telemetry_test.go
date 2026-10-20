package telemetry

import (
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)

func TestParseTopic(t *testing.T) {
	tests := []struct {
		name    string
		topic   string
		want    Topic
		wantErr bool
	}{
		{"valid", "org/demo-org/device/sim-001/telemetry", Topic{"demo-org", "sim-001"}, false},
		{"wrong suffix", "org/demo-org/device/sim-001/status", Topic{}, true},
		{"too short", "org/demo-org/telemetry", Topic{}, true},
		{"empty device", "org/demo-org/device//telemetry", Topic{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTopic(tt.topic)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseReading(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr bool
	}{
		{"valid", `{"temperature_c":21.5,"humidity_rh":55,"ts":"2026-10-13T11:59:00Z"}`, false},
		{"humidity too high", `{"temperature_c":21.5,"humidity_rh":150,"ts":"2026-10-13T11:59:00Z"}`, true},
		{"temperature too low", `{"temperature_c":-90,"humidity_rh":50,"ts":"2026-10-13T11:59:00Z"}`, true},
		{"missing field", `{"humidity_rh":50,"ts":"2026-10-13T11:59:00Z"}`, true},
		{"unknown field", `{"temperature_c":21,"humidity_rh":50,"ts":"2026-10-13T11:59:00Z","x":1}`, true},
		{"not json", `hello`, true},
		{"future timestamp", `{"temperature_c":21,"humidity_rh":50,"ts":"2026-10-13T13:00:00Z"}`, true},
		{"too large", `{"temperature_c":21,"humidity_rh":50,"ts":"2026-10-13T11:59:00Z","pad":"` + strings.Repeat("a", 2000) + `"}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseReading([]byte(tt.payload), now)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
