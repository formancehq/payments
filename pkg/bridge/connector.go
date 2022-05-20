package bridge

import (
	"context"
	"github.com/numary/payments/pkg"
)

type Connector[T payments.ConnectorConfigObject, S payments.ConnectorState] interface {
	ApplyDefaults(t T) T
	Name() string
	Start(ctx context.Context, object T, state S) error
	Stop(ctx context.Context) error
}
