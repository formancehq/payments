package api

import (
	"context"
	"net/http"

	"github.com/pkg/errors"

	"github.com/formancehq/payments/internal/app/models"

	"github.com/formancehq/payments/internal/app/storage"

	"github.com/formancehq/go-libs/sharedlogging"
	"github.com/formancehq/go-libs/sharedpublish"
	"github.com/formancehq/payments/internal/app/ingestion"
	"github.com/formancehq/payments/internal/app/integration"
	"github.com/formancehq/payments/internal/app/payments"
	"github.com/formancehq/payments/internal/app/task"
	"go.uber.org/dig"
	"go.uber.org/fx"
)

type connectorHandler struct {
	Handler  http.Handler
	Provider models.ConnectorProvider
}

func addConnector[
	ConnectorConfig payments.ConnectorConfigObject,
	TaskDescriptor payments.TaskDescriptor,
](loader integration.Loader[ConnectorConfig, TaskDescriptor],
) fx.Option {
	return fx.Options(
		fx.Provide(func(store *storage.Storage,
			publisher sharedpublish.Publisher,
		) *integration.ConnectorManager[ConnectorConfig, TaskDescriptor] {
			logger := sharedlogging.GetLogger(context.Background())

			schedulerFactory := integration.TaskSchedulerFactoryFn[TaskDescriptor](func(
				resolver task.Resolver[TaskDescriptor], maxTasks int,
			) *task.DefaultTaskScheduler[TaskDescriptor] {
				return task.NewDefaultScheduler[TaskDescriptor](loader.Name(), logger,
					store, task.ContainerFactoryFn(func(ctx context.Context,
						descriptor payments.TaskDescriptor,
					) (*dig.Container, error) {
						container := dig.New()

						if err := container.Provide(func() ingestion.Ingester {
							return ingestion.NewDefaultIngester(loader.Name(), descriptor, store,
								logger.WithFields(map[string]interface{}{
									"task-id": payments.IDFromDescriptor(descriptor),
								}), publisher)
						}); err != nil {
							return nil, err
						}

						return container, nil
					}), resolver, maxTasks)
			})

			return integration.NewConnectorManager[ConnectorConfig, TaskDescriptor](logger,
				store, loader, schedulerFactory)
		}),
		fx.Provide(fx.Annotate(func(cm *integration.ConnectorManager[ConnectorConfig,
			TaskDescriptor],
		) connectorHandler {
			return connectorHandler{
				Handler:  connectorRouter(loader.Name(), cm),
				Provider: loader.Name(),
			}
		}, fx.ResultTags(`group:"connectorHandlers"`))),
		fx.Invoke(func(lc fx.Lifecycle, cm *integration.ConnectorManager[ConnectorConfig, TaskDescriptor]) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					err := cm.Restore(ctx)
					if err != nil && !errors.Is(err, integration.ErrNotInstalled) {
						return err
					}

					return nil
				},
			})
		}),
	)
}
