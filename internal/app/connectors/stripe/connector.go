package stripe

import (
	"context"

	"github.com/formancehq/payments/internal/app/models"

	"github.com/formancehq/payments/internal/app/integration"
	"github.com/formancehq/payments/internal/app/task"

	"github.com/formancehq/go-libs/sharedlogging"
)

const Name = models.ConnectorProviderStripe

type Connector struct {
	logger sharedlogging.Logger
	cfg    Config
}

func (c *Connector) Install(ctx task.ConnectorContext[TaskDescriptor]) error {
	return ctx.Scheduler().Schedule(TaskDescriptor{
		Name: "Main task to periodically fetch transactions",
		Main: true,
	}, false)
}

func (c *Connector) Uninstall(ctx context.Context) error {
	return nil
}

func (c *Connector) Resolve(descriptor TaskDescriptor) task.Task {
	if descriptor.Main {
		return MainTask(c.cfg)
	}

	return ConnectedAccountTask(c.cfg, descriptor.Account)
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
