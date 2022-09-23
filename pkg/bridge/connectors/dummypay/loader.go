package dummypay

import (
	"github.com/numary/payments/pkg/bridge/integration"
)

func NewLoader() integration.Loader[Config, TaskDescriptor] {
	loader := integration.
		NewLoaderBuilder[Config, TaskDescriptor](connectorName).
		//WithLoad(func(logger sharedlogging.Logger, config Config) integration.Connector[TaskDescriptor] {
		//	return integration.NewConnectorBuilder[TaskDescriptor]().
		//		WithInstall(func(ctx task.ConnectorContext[TaskDescriptor]) error {
		//			return ctx.Scheduler().Schedule(newTaskReadFiles(), true)
		//		}).
		//		WithResolve(func(descriptor TaskDescriptor) task.Task {
		//			switch descriptor.Key {
		//			case taskKeyReadFiles:
		//				return taskReadFiles(config)
		//			case taskKeyIngest:
		//				return taskIngest(config, descriptor)
		//			}
		//
		//			return func() error {
		//				return fmt.Errorf("key '%s': %w", descriptor.Key, ErrMissingTask)
		//			}
		//		}).
		//		Build()
		//}).
		Build()

	return loader
}
