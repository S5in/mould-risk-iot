// Package risk holds the pure maths behind the mould-risk indicator.
// Nothing here talks to the network or the database, so it is easy to test.
package risk

import (
	"errors"
	"math"
)

// Magnus formula constants (Sonntag 1990), valid for air temperatures
// from -45 C to 60 C over liquid water.
const (
	magnusA = 17.62  // dimensionless
	magnusB = 243.12 // degrees C

	minTempC = -45.0
	maxTempC = 60.0
)

// ErrOutOfRange is returned when an input is outside the range the formula is valid for.
var ErrOutOfRange = errors.New("input outside the valid range for the dew point formula")

// DewPoint returns the dew point in degrees C for an air temperature in
// degrees C and a relative humidity in percent (greater than 0, up to 100).
//
// The dew point is the temperature a surface must fall to before water
// condenses on it. The closer a wall is to the dew point, the higher the
// condensation (and mould) risk.
func DewPoint(tempC, rh float64) (float64, error) {
	if math.IsNaN(tempC) || math.IsNaN(rh) || math.IsInf(tempC, 0) || math.IsInf(rh, 0) {
		return 0, ErrOutOfRange
	}
	// rh must be above 0 because ln(0) is undefined.
	if rh <= 0 || rh > 100 || tempC < minTempC || tempC > maxTempC {
		return 0, ErrOutOfRange
	}
	gamma := math.Log(rh/100) + magnusA*tempC/(magnusB+tempC)
	return magnusB * gamma / (magnusA - gamma), nil
}
