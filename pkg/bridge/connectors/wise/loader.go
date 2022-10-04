package wise

import (
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/integration"
	"github.com/numary/payments/pkg/bridge/task"
)

// NewLoader creates a new loader.
func NewLoader() integration.Loader[Config, TaskDefinition] {
	loader := integration.NewLoaderBuilder[Config, TaskDefinition]("wise").
		WithLoad(func(logger sharedlogging.Logger, config Config) integration.Connector[TaskDefinition] {
			return integration.NewConnectorBuilder[TaskDefinition]().
				WithInstall(func(ctx task.ConnectorContext[TaskDefinition]) error {
					return ctx.Scheduler().
						Schedule(
							TaskDefinition{Name: taskNameFetchProfiles},
							false)
				}).
				WithResolve(resolveTasks(logger, config)).
				Build()
		}).Build()

	return loader
}
