package metrics

import (
	"github.com/formancehq/payments/pkg/connector/metrics"
)

// NewHTTPClient creates a new HTTP client with metrics instrumentation.
var NewHTTPClient = metrics.NewHTTPClient
