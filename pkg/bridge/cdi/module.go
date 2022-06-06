package cdi

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/go-libs/sharedpublish"
	"github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/http"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/integration"
	"github.com/numary/payments/pkg/bridge/task"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/fx"
	http2 "net/http"
)

type ConnectorHandler struct {
	Handler http2.Handler
	Name    string
}

func ConnectorModule[
	ConnectorConfig payments.ConnectorConfigObject,
	TaskDescriptor payments.TaskDescriptor,
	TaskState any,
](useScopes bool, loader integration.Loader[ConnectorConfig, TaskDescriptor, TaskState]) fx.Option {
	return fx.Options(
		fx.Provide(func(db *mongo.Database, publisher sharedpublish.Publisher) *integration.ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState] {
			connectorStore := integration.NewMongoDBConnectorStore(db)
			taskStore := task.NewMongoDBStore[TaskDescriptor, TaskState](db)
			logger := sharedlogging.GetLogger(context.Background())
			schedulerFactory := integration.TaskSchedulerFactoryFn[TaskDescriptor, TaskState](func(resolver task.Resolver[TaskDescriptor, TaskState], maxTasks int) *task.DefaultTaskScheduler[TaskDescriptor, TaskState] {
				return task.NewDefaultScheduler[TaskDescriptor, TaskState](loader.Name(), logger, taskStore, task.IngesterFactoryFn(func(ctx context.Context, provider string, descriptor payments.TaskDescriptor) ingestion.Ingester {
					return ingestion.NewDefaultIngester(provider, descriptor, db, logger.WithFields(map[string]interface{}{
						"task-id": payments.IDFromDescriptor(descriptor),
					}), publisher)
				}), resolver, maxTasks)
			})
			return integration.NewConnectorManager[ConnectorConfig, TaskDescriptor, TaskState](logger, connectorStore, loader, schedulerFactory)
		}),
		fx.Provide(fx.Annotate(func(cm *integration.ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]) ConnectorHandler {
			return ConnectorHandler{
				Handler: http.ConnectorRouter(loader.Name(), useScopes, cm),
				Name:    loader.Name(),
			}
		}, fx.ResultTags(`group:"connectorHandlers"`))),
		fx.Invoke(func(lc fx.Lifecycle, cm *integration.ConnectorManager[ConnectorConfig, TaskDescriptor, TaskState]) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					_ = cm.Restore(ctx)
					return nil
				},
			})
		}),
	)
}
