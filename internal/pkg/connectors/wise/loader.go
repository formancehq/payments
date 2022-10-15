package wise

import (
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/internal/pkg/integration"
	"github.com/numary/payments/internal/pkg/task"
)

// NewLoader creates a new loader.
func NewLoader() integration.Loader[Config, TaskDescriptor] {
	loader := integration.NewLoaderBuilder[Config, TaskDescriptor]("wise").
		WithLoad(func(logger sharedlogging.Logger, config Config) integration.Connector[TaskDescriptor] {
			return integration.NewConnectorBuilder[TaskDescriptor]().
				WithInstall(func(ctx task.ConnectorContext[TaskDescriptor]) error {
					return ctx.Scheduler().
						Schedule(
							TaskDescriptor{Name: taskNameFetchProfiles},
							false)
				}).
				WithResolve(resolveTasks(logger, config)).
				Build()
		}).Build()

	return loader
}
