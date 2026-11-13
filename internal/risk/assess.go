package risk

import (
	"errors"
	"fmt"
)

// Level is the mould-risk level for one reading.
type Level int

const (
	LevelLow Level = iota
	LevelMedium
	LevelHigh
)

func (l Level) String() string {
	switch l {
	case LevelLow:
		return "low"
	case LevelMedium:
		return "medium"
	case LevelHigh:
		return "high"
	default:
		return "unknown"
	}
}

// Thresholds are the configurable numbers behind the risk level.
//
// Where the default values come from:
//   - RHMedium 60: Irish guidance gives "less than 60%" (HSA Mould Risk
//     Assessment Template No.30) and "below 60% relative humidity" (State
//     Claims Agency guidelines, 2006).
//   - RHHigh 70: SEAI S.R. 54.
//   - SurfaceMarginMediumC 2: a precautionary buffer chosen by the author, not a standard.
//   - ColdSurfaceOffsetC 3: an assumption for an unmeasured cold surface, not a standard.
type Thresholds struct {
	RHMedium float64 // relative humidity (%) at or above which the risk is at least medium
	RHHigh   float64 // relative humidity (%) at or above which the risk is high

	// Air temperature minus the assumed temperature of the coldest wall surface.
	// The sensor measures the air, not the wall, so this is an assumption.
	ColdSurfaceOffsetC float64
	// Medium risk when the coldest surface is within this many degrees C of the dew point.
	SurfaceMarginMediumC float64
}

// DefaultThresholds returns the development defaults.
func DefaultThresholds() Thresholds {
	return Thresholds{
		RHMedium:             60,
		RHHigh:               70,
		ColdSurfaceOffsetC:   3,
		SurfaceMarginMediumC: 2,
	}
}

// ErrBadThresholds is returned when the thresholds do not make sense together.
var ErrBadThresholds = errors.New("invalid risk thresholds")

// Validate checks the thresholds are sensible.
func (t Thresholds) Validate() error {
	if t.RHMedium <= 0 || t.RHHigh > 100 || t.RHMedium >= t.RHHigh {
		return fmt.Errorf("%w: need 0 < RHMedium < RHHigh <= 100", ErrBadThresholds)
	}
	if t.ColdSurfaceOffsetC < 0 || t.SurfaceMarginMediumC < 0 {
		return fmt.Errorf("%w: offsets must not be negative", ErrBadThresholds)
	}
	return nil
}

// Assessment is the risk level plus everything needed to explain it.
type Assessment struct {
	Level          Level
	DewPointC      float64
	SurfaceMarginC float64  // assumed coldest surface temperature minus dew point
	Reasons        []string // plain-language reasons, shown to the user
}

// Assess works out the risk level for one reading. The level is the higher of
// two checks: relative humidity, and how close the assumed coldest surface is
// to the dew point (condensation happens when it reaches it).
//
// This is an indicator for one moment in time, not a diagnosis. Mould depends
// on how long conditions last, which a later step will add.
func Assess(tempC, rh float64, th Thresholds) (Assessment, error) {
	if err := th.Validate(); err != nil {
		return Assessment{}, err
	}
	dp, err := DewPoint(tempC, rh)
	if err != nil {
		return Assessment{}, err
	}
	margin := tempC - th.ColdSurfaceOffsetC - dp

	a := Assessment{Level: LevelLow, DewPointC: dp, SurfaceMarginC: margin}
	raise := func(l Level, reason string) {
		if l > a.Level {
			a.Level = l
		}
		a.Reasons = append(a.Reasons, reason)
	}

	switch {
	case rh >= th.RHHigh:
		raise(LevelHigh, fmt.Sprintf("humidity %.1f%% is at or above the high threshold of %.0f%%", rh, th.RHHigh))
	case rh >= th.RHMedium:
		raise(LevelMedium, fmt.Sprintf("humidity %.1f%% is at or above the medium threshold of %.0f%%", rh, th.RHMedium))
	}
	switch {
	case margin <= 0:
		raise(LevelHigh, fmt.Sprintf("a wall %.0f C colder than the room would be at or below the dew point of %.1f C, so water would condense", th.ColdSurfaceOffsetC, dp))
	case margin <= th.SurfaceMarginMediumC:
		raise(LevelMedium, fmt.Sprintf("a wall %.0f C colder than the room would be within %.1f C of the dew point of %.1f C", th.ColdSurfaceOffsetC, th.SurfaceMarginMediumC, dp))
	}
	if len(a.Reasons) == 0 {
		a.Reasons = append(a.Reasons, "humidity and the dew point margin are both below their thresholds")
	}
	return a, nil
}
