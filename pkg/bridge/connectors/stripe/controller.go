package stripe

import (
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
)

type Controller struct{}

func (c *Controller) New(logObjectStorage bridge.LogObjectStorage, logger sharedlogging.Logger, ingester bridge.Ingester[Config, State, *Connector]) (*Connector, error) {
	return NewConnector(logObjectStorage, logger, ingester), nil
}

const (
	ConnectorName = "stripe"
)
