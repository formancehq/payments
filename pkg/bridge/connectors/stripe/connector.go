package stripe

import (
	"context"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/integration"
	"github.com/numary/payments/pkg/bridge/task"
)

const connectorName = "stripe"

type Connector struct {
	logger sharedlogging.Logger
	client Client
	cfg    Config
}

func (c *Connector) Install(ctx task.ConnectorContext[TaskDescriptor]) error {
	return ctx.Scheduler().Schedule(TaskDescriptor{
		Main: true,
	}, false)
}

func (c *Connector) Uninstall(ctx context.Context) error {
	return nil
}

func (c *Connector) Resolve(descriptor TaskDescriptor) task.Task {
	if descriptor.Main {
		return MainTask(c.client, c.cfg)
	}
	return ConnectedAccountTask(c.client, c.cfg, descriptor.Account)
}

var _ integration.Connector[TaskDescriptor] = &Connector{}

func NewConnector(logger sharedlogging.Logger, client Client, cfg Config) *Connector {
	return &Connector{
		logger: logger.WithFields(map[string]any{
			"component": "connector",
		}),
		client: client,
		cfg:    cfg,
	}
}
