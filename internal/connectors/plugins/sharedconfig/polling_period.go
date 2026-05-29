package sharedconfig

import (
	"encoding/json"
	"sync/atomic"
	"time"
)

var (
	// these have no hard-coded default value because they are supposed to be set as part of the server configuration
	// using command line flags. as such their default values are also configured in the cmd package
	minimumPollingPeriod atomic.Int64
	defaultPollingPeriod atomic.Int64
)

func GetMinimumPollingPeriod() time.Duration { return time.Duration(minimumPollingPeriod.Load()) }
func GetDefaultPollingPeriod() time.Duration { return time.Duration(defaultPollingPeriod.Load()) }

// SetPollingPeriodDefaults is only intended to be called from connectors.Manager
// which gets its configuration from command line arguments set by the service administrator
func SetPollingPeriodDefaults(def, min time.Duration) {
	defaultPollingPeriod.Store(int64(def))
	minimumPollingPeriod.Store(int64(min))
}

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
