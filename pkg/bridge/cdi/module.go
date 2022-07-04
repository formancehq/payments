package cdi

import (
	"context"
	http2 "net/http"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/go-libs/sharedpublish"
	"github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/http"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/integration"
	"github.com/numary/payments/pkg/bridge/task"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/dig"
	"go.uber.org/fx"
)

type ConnectorHandler struct {
	Handler http2.Handler
	Name    string
}

func ConnectorModule[
	ConnectorConfig payments.ConnectorConfigObject,
	TaskDescriptor payments.TaskDescriptor,
](useScopes bool, loader integration.Loader[ConnectorConfig, TaskDescriptor]) fx.Option {
	return fx.Options(
		fx.Provide(func(db *mongo.Database, publisher sharedpublish.Publisher) *integration.ConnectorManager[ConnectorConfig, TaskDescriptor] {
			connectorStore := integration.NewMongoDBConnectorStore(db)
			taskStore := task.NewMongoDBStore[TaskDescriptor](db)
			logger := sharedlogging.GetLogger(context.Background())
			schedulerFactory := integration.TaskSchedulerFactoryFn[TaskDescriptor](func(resolver task.Resolver[TaskDescriptor], maxTasks int) *task.DefaultTaskScheduler[TaskDescriptor] {
				return task.NewDefaultScheduler[TaskDescriptor](loader.Name(), logger, taskStore, task.ContainerFactoryFn(func(ctx context.Context, descriptor payments.TaskDescriptor) (*dig.Container, error) {
					container := dig.New()
					if err := container.Provide(func() ingestion.Ingester {
						return ingestion.NewDefaultIngester(loader.Name(), descriptor, db, logger.WithFields(map[string]interface{}{
							"task-id": payments.IDFromDescriptor(descriptor),
						}), publisher)
					}); err != nil {
						return nil, err
					}
					return container, nil
				}), resolver, maxTasks)
			})
			return integration.NewConnectorManager[ConnectorConfig, TaskDescriptor](logger, connectorStore, loader, schedulerFactory)
		}),
		fx.Provide(fx.Annotate(func(cm *integration.ConnectorManager[ConnectorConfig, TaskDescriptor]) ConnectorHandler {
			return ConnectorHandler{
				Handler: http.ConnectorRouter(loader.Name(), useScopes, cm),
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
