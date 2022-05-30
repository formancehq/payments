package stripe

import (
	"context"
	"github.com/alitto/pond"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
	"net/http"
	"time"
)

const connectorName = "stripe"

type Connector struct {
	logObjectStorage bridge.LogObjectStorage
	runner           *Runner
	logger           sharedlogging.Logger
	scheduler        *Scheduler
	ingester         bridge.Ingester[State]
	pool             *pond.WorkerPool
	client           *defaultClient
}

func (c *Connector) Name() string {
	return connectorName
}

func (c *Connector) Start(ctx context.Context, cfg Config, state State) error {
	c.logger.WithFields(map[string]interface{}{
		"pool":           cfg.Pool,
		"page-size":      cfg.PageSize,
		"polling-period": cfg.PollingPeriod,
	}).Infof("Starting connector")
	c.pool = pond.New(cfg.Pool, 0)
	c.client = NewDefaultClient(http.DefaultClient, c.pool, cfg.ApiKey)
	c.scheduler = NewScheduler(c.logObjectStorage, c.logger.WithFields(map[string]interface{}{
		"component": "scheduler",
	}), c.ingester, c.client, cfg, state)
	return c.scheduler.Start(ctx)
}

func (c *Connector) Stop(ctx context.Context) error {
	if c.scheduler != nil {
		err := c.scheduler.Stop(ctx)
		if err != nil {
			return err
		}
		c.scheduler = nil
	}
	if c.pool != nil {
		c.pool.StopAndWait()
		c.pool = nil
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
