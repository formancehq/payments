package api

import (
	"context"
	"net/http"

	"github.com/formancehq/go-libs/sharedlogging"
	"github.com/formancehq/go-libs/sharedpublish"
	"github.com/formancehq/payments/internal/app/ingestion"
	"github.com/formancehq/payments/internal/app/integration"
	"github.com/formancehq/payments/internal/app/payments"
	"github.com/formancehq/payments/internal/app/task"
	"github.com/formancehq/payments/internal/pkg/writeonly"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/dig"
	"go.uber.org/fx"
)

type connectorHandler struct {
	Handler http.Handler
	Name    string
}

func addConnector[
	ConnectorConfig payments.ConnectorConfigObject,
	TaskDescriptor payments.TaskDescriptor,
](loader integration.Loader[ConnectorConfig, TaskDescriptor],
) fx.Option {
	return fx.Options(
		fx.Provide(func(db *mongo.Database,
			publisher sharedpublish.Publisher,
		) *integration.ConnectorManager[ConnectorConfig, TaskDescriptor] {
			connectorStore := integration.NewMongoDBConnectorStore(db)
			taskStore := task.NewMongoDBStore[TaskDescriptor](db)
			logger := sharedlogging.GetLogger(context.Background())

			schedulerFactory := integration.TaskSchedulerFactoryFn[TaskDescriptor](func(
				resolver task.Resolver[TaskDescriptor], maxTasks int,
			) *task.DefaultTaskScheduler[TaskDescriptor] {
				return task.NewDefaultScheduler[TaskDescriptor](loader.Name(), logger,
					taskStore, task.ContainerFactoryFn(func(ctx context.Context,
						descriptor payments.TaskDescriptor,
					) (*dig.Container, error) {
						container := dig.New()

						if err := container.Provide(func() ingestion.Ingester {
							return ingestion.NewDefaultIngester(loader.Name(), descriptor, db,
								logger.WithFields(map[string]interface{}{
									"task-id": payments.IDFromDescriptor(descriptor),
								}), publisher)
						}); err != nil {
							return nil, err
						}

						err := container.Provide(func() writeonly.Storage {
							return writeonly.NewMongoDBStorage(db, loader.Name(), descriptor)
						})
						if err != nil {
							panic(err)
						}

						return container, nil
					}), resolver, maxTasks)
			})

			return integration.NewConnectorManager[ConnectorConfig, TaskDescriptor](logger,
				connectorStore, loader, schedulerFactory)
		}),
		fx.Provide(fx.Annotate(func(cm *integration.ConnectorManager[ConnectorConfig,
			TaskDescriptor],
		) connectorHandler {
			return connectorHandler{
				Handler: connectorRouter(loader.Name(), cm),
				Name:    loader.Name(),
			}
		}, fx.ResultTags(`group:"connectorHandlers"`))),
		fx.Invoke(func(lc fx.Lifecycle, cm *integration.ConnectorManager[ConnectorConfig, TaskDescriptor]) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					_ = cm.Restore(ctx)

					return nil
				},
			})
		}),
	)
}
