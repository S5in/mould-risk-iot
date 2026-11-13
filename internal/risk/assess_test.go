package risk

import (
	"errors"
	"testing"
)

func TestAssess(t *testing.T) {
	def := DefaultThresholds()
	// Thresholds that make the humidity checks hard to trigger, so the
	// cold-wall check decides on its own.
	wallOnly := Thresholds{RHMedium: 90, RHHigh: 95, ColdSurfaceOffsetC: 8, SurfaceMarginMediumC: 2}

	tests := []struct {
		name  string
		tempC float64
		rh    float64
		th    Thresholds
		want  Level
	}{
		{name: "dry room is low", tempC: 20, rh: 45, th: def, want: LevelLow},
		{name: "reading from esp32-001 is low", tempC: 16.4, rh: 58.7, th: def, want: LevelLow},
		{name: "humidity at medium threshold", tempC: 20, rh: 60, th: def, want: LevelMedium},
		{name: "humidity at high threshold", tempC: 20, rh: 70, th: def, want: LevelHigh},
		{name: "very humid is high", tempC: 20, rh: 90, th: def, want: LevelHigh},
		{name: "cold wall check alone gives high", tempC: 20, rh: 65, th: wallOnly, want: LevelHigh},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Assess(tt.tempC, tt.rh, tt.th)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Level != tt.want {
				t.Errorf("Assess(%.1f, %.1f) level = %s, want %s (reasons: %v)", tt.tempC, tt.rh, got.Level, tt.want, got.Reasons)
			}
			if len(got.Reasons) == 0 {
				t.Error("every assessment must explain itself")
			}
		})
	}
}

func TestAssessRejectsBadInput(t *testing.T) {
	if _, err := Assess(20, 0, DefaultThresholds()); !errors.Is(err, ErrOutOfRange) {
		t.Errorf("bad reading: got %v, want ErrOutOfRange", err)
	}
	bad := DefaultThresholds()
	bad.RHMedium, bad.RHHigh = 80, 70
	if _, err := Assess(20, 50, bad); !errors.Is(err, ErrBadThresholds) {
		t.Errorf("bad thresholds: got %v, want ErrBadThresholds", err)
	}
}

func TestLevelString(t *testing.T) {
	if LevelLow.String() != "low" || LevelMedium.String() != "medium" || LevelHigh.String() != "high" {
		t.Error("unexpected level names")
	}
}
