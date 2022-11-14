package stripe

import (
	"context"

	"github.com/numary/payments/internal/pkg/integration"
	"github.com/numary/payments/internal/pkg/task"

	"github.com/numary/go-libs/sharedlogging"
)

const connectorName = "stripe"

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
