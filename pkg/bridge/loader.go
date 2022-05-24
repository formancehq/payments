package bridge

import (
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg"
)

type Loader[
	CONFIG payments.ConnectorConfigObject,
	STATE payments.ConnectorState,
	CONNECTOR Connector[CONFIG, STATE],
] interface {
	Load(logObjectStore LogObjectStorage, logger sharedlogging.Logger, ingester Ingester[STATE]) (CONNECTOR, error)
}

type LoaderFn[
	T payments.ConnectorConfigObject,
	S payments.ConnectorState,
	C Connector[T, S],
] func(logObjectStore LogObjectStorage, logger sharedlogging.Logger, ingester Ingester[S]) (C, error)

func (fn LoaderFn[T, S, C]) Load(
	logObjectStore LogObjectStorage, logger sharedlogging.Logger, ingester Ingester[S],
) (C, error) {
	return fn(logObjectStore, logger, ingester)
}
