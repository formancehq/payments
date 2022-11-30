package currencycloud

import (
	"context"

	"github.com/formancehq/go-libs/sharedlogging"
	"github.com/formancehq/payments/internal/pkg/integration"
	"github.com/formancehq/payments/internal/pkg/task"
)

const Name = "currencycloud"

type Connector struct {
	logger sharedlogging.Logger
	cfg    Config
}

func (c *Connector) Install(ctx task.ConnectorContext[TaskDescriptor]) error {
	return ctx.Scheduler().Schedule(TaskDescriptor{Name: taskNameFetchTransactions}, true)
}

func (c *Connector) Uninstall(ctx context.Context) error {
	return nil
}

func (c *Connector) Resolve(descriptor TaskDescriptor) task.Task {
	return resolveTasks(c.logger, c.cfg)
}

var _ integration.Connector[TaskDescriptor] = &Connector{}

func newConnector(logger sharedlogging.Logger, cfg Config) *Connector {
	return &Connector{
		logger: logger.WithFields(map[string]any{
			"component": "connector",
		}),
		cfg: cfg,
	}
}
