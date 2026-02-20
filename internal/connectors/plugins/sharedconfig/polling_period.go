package sharedconfig

import (
	"time"

	"github.com/formancehq/payments/pkg/connector"
)

// Polling period aliases for backward compatibility.
// The canonical implementations now live in pkg/connector.

const (
	MinimumPollingPeriod = connector.MinimumPollingPeriod
	DefaultPollingPeriod = connector.DefaultPollingPeriod
)

// PollingPeriod is an alias to pkg/connector.PollingPeriod.
type PollingPeriod = connector.PollingPeriod

// NewPollingPeriod is an alias to pkg/connector.NewPollingPeriod.
func NewPollingPeriod(raw string, def, min time.Duration) (PollingPeriod, error) {
	return connector.NewPollingPeriod(raw, def, min)
}
