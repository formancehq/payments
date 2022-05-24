package stripe

import (
	"context"
	"github.com/alitto/pond"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge"
	"github.com/stripe/stripe-go/v72"
	"sync"
	"time"
)

type Scheduler struct {
	logObjectStorage bridge.LogObjectStorage
	runner           *Runner
	accountRunners   map[string]*Runner
	logger           sharedlogging.Logger
	ingester         bridge.Ingester[State]
	pool             *pond.WorkerPool
	mu               sync.RWMutex
}

func (c *Scheduler) Name() string {
	return "stripe"
}

func (c *Scheduler) createRunner(account string, cfg Config, state TimelineState) {

	c.mu.Lock()
	defer c.mu.Unlock()

	logger := c.logger.WithFields(map[string]interface{}{
		"account": account,
	})

	logger.Infof("Create new runner")

	runner := NewRunner(
		logger.WithFields(map[string]interface{}{
			"component": "runner",
		}),
		NewDefaultIngester(c.Name(), account, logger.WithFields(map[string]interface{}{
			"component": "ingester",
		}), c.ingester, c.logObjectStorage),
		NewTimeline(c.pool, cfg, state, WithTimelineExpand("data.source")),
	)
	c.accountRunners[account] = runner

	go func(runner *Runner) {
		err := runner.Run(context.Background())
		if err != nil {
			panic(err)
		}
	}(runner)
}

func (c *Scheduler) wrapMainIngester(cfg Config, i *defaultIngester) IngesterFn {
	return func(ctx context.Context, batch []stripe.BalanceTransaction, commitState TimelineState, tail bool) error {
		err := i.Ingest(ctx, batch, commitState, tail)
		if err != nil {
			return err
		}
		missingAccounts := make([]string, 0)
		func() {
			c.mu.RLock()
			defer c.mu.RUnlock()

			for _, tx := range batch {
				if tx.Type == "transfer" {
					accountId := tx.Source.Transfer.Destination.ID
					_, exists := c.accountRunners[accountId]
					if !exists {
						missingAccounts = append(missingAccounts, accountId)
					}
				}
			}
		}()
		if len(missingAccounts) > 0 {
			for _, account := range missingAccounts {
				c.createRunner(account, cfg, TimelineState{})
			}
		}
		return nil
	}
}

func (c *Scheduler) Start(ctx context.Context, cfg Config, state State) error {
	c.pool = pond.New(cfg.Pool, 0)

	c.runner = NewRunner(
		c.logger.WithFields(map[string]interface{}{
			"component": "runner",
			"timeline":  "main",
		}),
		c.wrapMainIngester(cfg, NewDefaultIngester(c.Name(), "", c.logger.WithFields(map[string]interface{}{
			"component": "ingester",
			"timeline":  "main",
		}), c.ingester, c.logObjectStorage)),
		NewTimeline(c.pool, cfg, state.TimelineState, WithTimelineExpand("data.source")),
	)

	go func() {
		err := c.runner.Run(ctx)
		if err != nil {
			panic(err)
		}
	}()

	if state.Accounts != nil {
		for account, accountState := range state.Accounts {
			c.createRunner(account, cfg, accountState)
		}
	}

	return nil
}

func (c *Scheduler) Stop(ctx context.Context) error {
	if c.runner != nil {
		err := c.runner.Stop(ctx)
		if err != nil {
			return err
		}
		c.runner = nil
	}
	for account, runner := range c.accountRunners {
		err := runner.Stop(ctx)
		if err != nil {
			return err
		}
		delete(c.accountRunners, account)
	}
	if c.pool != nil {
		c.pool.StopAndWait()
		c.pool = nil
	}
	return nil
}

func (c *Scheduler) ApplyDefaults(cfg Config) Config {
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

func NewScheduler(logObjectStorage bridge.LogObjectStorage, logger sharedlogging.Logger, ingester bridge.Ingester[State]) *Scheduler {
	return &Scheduler{
		logObjectStorage: logObjectStorage,
		logger:           logger,
		ingester:         ingester,
		accountRunners:   map[string]*Runner{},
	}
}
