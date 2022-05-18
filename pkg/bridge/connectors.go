package bridge

import (
	"context"
)

type ConnectorConfigObject interface {
	Validate() error
}

type ConnectorState interface{}

type Connector[T ConnectorConfigObject, S ConnectorState] interface {
	ApplyDefaults(t T) T
	Name() string
	Start(ctx context.Context, object T, state S) error
	Stop(ctx context.Context) error
}
