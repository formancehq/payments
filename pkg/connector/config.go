package connector

import (
	"encoding/json"
	"time"
)

const (
	MinimumPollingPeriod = 20 * time.Minute
	DefaultPollingPeriod = 30 * time.Minute
)

// PollingPeriod represents a duration used for polling intervals in connectors.
type PollingPeriod time.Duration

// Duration returns the underlying time.Duration value.
func (p PollingPeriod) Duration() time.Duration { return time.Duration(p) }

// MarshalJSON implements the json.Marshaler interface.
func (p PollingPeriod) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(p).String())
}

// NewPollingPeriod creates a new PollingPeriod from a raw string value,
// applying minimum and default constraints.
func NewPollingPeriod(raw string, def, min time.Duration) (PollingPeriod, error) {
	if raw == "" {
		if def < min {
			return PollingPeriod(min), nil
		}
		return PollingPeriod(def), nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, err
	}
	if v < min {
		v = min
	}
	return PollingPeriod(v), nil
}
