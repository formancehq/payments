package stripe

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
	"time"
)

type Connector struct {
	logObjectStorage bridge.LogObjectStorage
	runner           *Runner
	logger           sharedlogging.Logger
	ingester         bridge.Ingester[State]
	scheduler        *Scheduler
}

func (c *Connector) Name() string {
	return "stripe"
}

func (c *Connector) Start(ctx context.Context, cfg Config, state State) error {
	c.scheduler = NewScheduler(c.logObjectStorage, c.logger.WithFields(map[string]interface{}{
		"component": "scheduler",
	}), c.ingester)
	return c.scheduler.Start(ctx, cfg, state)
}

func (c *Connector) Stop(ctx context.Context) error {
	if c.scheduler != nil {
		err := c.scheduler.Stop(ctx)
		if err != nil {
			return err
		}
		c.scheduler = nil
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

func NewConnector(logObjectStorage bridge.LogObjectStorage, logger sharedlogging.Logger, ingester bridge.Ingester[State]) (*Connector, error) {
	return &Connector{
		logObjectStorage: logObjectStorage,
		logger:           logger,
		ingester:         ingester,
	}, nil
}
