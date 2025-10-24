package sharedconfig

import (
	"encoding/json"
	"time"
)

const (
	MinimumPollingPeriod = 20 * time.Minute
	DefaultPollingPeriod = 30 * time.Minute
)

type PollingPeriod time.Duration

func (p PollingPeriod) Duration() time.Duration { return time.Duration(p) }

func (p PollingPeriod) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(p).String())
}

// Helper to construct the value while applying min/default.
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
