package bridge

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
)

type LogObjectStorage interface {
	Store(ctx context.Context, objects ...any) error
}

type Controller[
	T ConnectorConfigObject,
	S ConnectorState,
	C Connector[T, S],
] interface {
	New(logObjectStore LogObjectStorage, logger sharedlogging.Logger, ingester Ingester[T, S, C]) (C, error)
}
