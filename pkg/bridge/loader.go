package bridge

import (
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg"
)

type Loader[
	T payments.ConnectorConfigObject,
	S payments.ConnectorState,
	C Connector[T, S],
] interface {
	Load(logObjectStore LogObjectStorage, logger sharedlogging.Logger, ingester Ingester[T, S, C]) (C, error)
}

type LoaderFn[
	T payments.ConnectorConfigObject,
	S payments.ConnectorState,
	C Connector[T, S],
] func(logObjectStore LogObjectStorage, logger sharedlogging.Logger, ingester Ingester[T, S, C]) (C, error)

func (fn LoaderFn[T, S, C]) Load(
	logObjectStore LogObjectStorage, logger sharedlogging.Logger, ingester Ingester[T, S, C],
) (C, error) {
	return fn(logObjectStore, logger, ingester)
}
