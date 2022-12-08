package modulr

import (
	"context"

	"github.com/formancehq/go-libs/sharedlogging"
	"github.com/formancehq/payments/internal/app/integration"
	"github.com/formancehq/payments/internal/app/task"
)

const Name = "modulr"

type Connector struct {
	logger sharedlogging.Logger
	cfg    Config
}

func (c *Connector) Install(ctx task.ConnectorContext[TaskDescriptor]) error {
	return ctx.Scheduler().Schedule(TaskDescriptor{
		Name: "Fetch accounts from client",
		Key:  taskNameFetchAccounts,
	}, false)
}

func (c *Connector) Uninstall(ctx context.Context) error {
	return nil
}

func (c *Connector) Resolve(descriptor TaskDescriptor) task.Task {
	return resolveTasks(c.logger, c.cfg)(descriptor)
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
