package stripe

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type Connector struct {
	db       *mongo.Database
	runner   *Runner
	logger   sharedlogging.Logger
	ingester bridge.Ingester[Config, State, *Connector]
}

func (c *Connector) Name() string {
	return ConnectorName
}

func (c *Connector) Start(ctx context.Context, object Config, state State) error {
	c.runner = NewRunner(c.db, c.logger, c.ingester, object, state)
	return c.runner.Run(ctx)
}

func (c *Connector) Stop(ctx context.Context) error {
	if c.runner != nil {
		return c.runner.Stop(ctx)
	}
	return nil
}

func (c *Connector) ApplyDefaults(cfg Config) Config {
	if cfg.Pool == 0 {
		cfg.Pool = 1
	}
	if cfg.PageSize == 0 {
		cfg.PageSize = 100
	}
	if cfg.PollingPeriod == 0 {
		cfg.PollingPeriod = 5 * time.Second
	}
	return cfg
}

var _ bridge.Connector[Config, State] = &Connector{}

func NewConnector(db *mongo.Database, logger sharedlogging.Logger, ingester bridge.Ingester[Config, State, *Connector]) *Connector {
	return &Connector{
		db:       db,
		logger:   logger,
		ingester: ingester,
	}
}
