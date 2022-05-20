package noop

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
)

type Connector struct {
	logger sharedlogging.Logger
}

func (c *Connector) Name() string {
	return ConnectorName
}

func (c *Connector) Start(ctx context.Context, object Config, state State) error {
	c.logger.Info("Starting noop connector")
	return nil
}

func (c *Connector) Stop(ctx context.Context) error {
	c.logger.Info("Stopping noop connector")
	return nil
}

func (c *Connector) ApplyDefaults(cfg Config) Config {
	return cfg
}

var _ bridge.Connector[Config, State] = &Connector{}

func NewConnector(logger sharedlogging.Logger) *Connector {
	return &Connector{
		logger: logger,
	}
}
