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

func (c *Connector) Resolve(descriptor TaskDescriptor) task.Task[TaskDescriptor, TimelineState] {
	if descriptor.Main {
		return &mainTask{
			config: c.cfg,
			client: c.client,
		}
	}
	return &connectedAccountTask{
		account: descriptor.Account,
		client:  c.client,
		config:  c.cfg.TimelineConfig,
	}
}

var _ integration.Connector[TaskDescriptor, TimelineState] = &Connector{}

func NewConnector(logger sharedlogging.Logger, client Client, cfg Config) *Connector {
	return &Connector{
		logger: logger.WithFields(map[string]any{
			"component": "connector",
		}),
		client: client,
		cfg:    cfg,
	}
}
