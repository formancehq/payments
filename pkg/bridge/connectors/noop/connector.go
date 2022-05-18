package noop

import (
	"context"
	"fmt"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
	"go.mongodb.org/mongo-driver/mongo"
)

type Connector struct{}

func (c *Connector) Name() string {
	return ConnectorName
}

func (c *Connector) Start(ctx context.Context, object Config, state State) error {
	return fmt.Errorf("noop connector")
}

func (c *Connector) Stop(ctx context.Context) error {
	return nil
}

func (c *Connector) ApplyDefaults(cfg Config) Config {
	return cfg
}

var _ bridge.Connector[Config, State] = &Connector{}

func NewConnector(db *mongo.Database, logger sharedlogging.Logger, ingester bridge.Ingester[Config, State, *Connector]) *Connector {
	return &Connector{}
}
