package risk

import (
	"math"
	"testing"
)

func TestDewPoint(t *testing.T) {
	tests := []struct {
		name    string
		tempC   float64
		rh      float64
		want    float64
		wantErr bool
	}{
		{name: "20C 50%", tempC: 20, rh: 50, want: 9.26},
		{name: "saturated air: dew point equals air temperature", tempC: 20, rh: 100, want: 20},
		{name: "warm and humid", tempC: 25, rh: 80, want: 21.31},
		{name: "freezing air", tempC: 0, rh: 50, want: -9.20},
		{name: "reading from esp32-001", tempC: 16.4, rh: 58.7, want: 8.29},
		{name: "zero humidity is invalid", tempC: 20, rh: 0, wantErr: true},
		{name: "humidity above 100 is invalid", tempC: 20, rh: 101, wantErr: true},
		{name: "temperature too high", tempC: 70, rh: 50, wantErr: true},
		{name: "temperature too low", tempC: -50, rh: 50, wantErr: true},
		{name: "NaN humidity", tempC: 20, rh: math.NaN(), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DewPoint(tt.tempC, tt.rh)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %.2f", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(got-tt.want) > 0.05 {
				t.Errorf("DewPoint(%.1f, %.1f) = %.2f, want %.2f", tt.tempC, tt.rh, got, tt.want)
			}
		})
	}
}

// The dew point can never be above the air temperature.
func TestDewPointNeverAboveAirTemperature(t *testing.T) {
	for temp := -20.0; temp <= 40; temp += 5 {
		for rh := 5.0; rh <= 100; rh += 5 {
			got, err := DewPoint(temp, rh)
			if err != nil {
				t.Fatalf("unexpected error at %.0f C %.0f%%: %v", temp, rh, err)
			}
			if got > temp+1e-9 {
				t.Errorf("dew point %.2f above air temperature %.2f at %.0f%%", got, temp, rh)
			}
		}
	}
}
