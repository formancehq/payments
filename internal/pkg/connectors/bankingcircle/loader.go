package bankingcircle

import (
	"github.com/formancehq/go-libs/sharedlogging"
	"github.com/formancehq/payments/internal/pkg/integration"
	"github.com/formancehq/payments/internal/pkg/task"
)

const Name = "bankingcircle"

// NewLoader creates a new loader.
func NewLoader() integration.Loader[Config, TaskDescriptor] {
	loader := integration.NewLoaderBuilder[Config, TaskDescriptor](Name).
		WithLoad(func(logger sharedlogging.Logger, config Config) integration.Connector[TaskDescriptor] {
			return integration.NewConnectorBuilder[TaskDescriptor]().
				WithInstall(func(ctx task.ConnectorContext[TaskDescriptor]) error {
					return ctx.Scheduler().
						Schedule(TaskDescriptor{
							Name: "Fetch payments from source",
							Key:  taskNameFetchPayments,
						}, false)
				}).
				WithResolve(resolveTasks(logger, config)).
				Build()
		}).Build()

	return loader
}
