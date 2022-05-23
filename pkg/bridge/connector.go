package bridge

import (
	"context"
	"github.com/numary/payments/pkg"
)

// Connector provide entry point to a payment provider
// It requires a payments.ConnectorConfigObject representing the configuration of the specific payment provider
// as well as a payments.ConnectorState object which represents the state of the connector
type Connector[T payments.ConnectorConfigObject, S payments.ConnectorState] interface {
	// ApplyDefaults is used to fill default values of the provided configuration object
	ApplyDefaults(t T) T
	// Name
	Name() string
	// Start is used to start the connector. The implementation if in charge of starting all required resources.
	Start(ctx context.Context, object T, state S) error
	// Stop is used to stop the connector. It has to close all related resources opened by the connector.
	Stop(ctx context.Context) error
}
