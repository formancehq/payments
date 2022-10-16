package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"

	"github.com/numary/payments/internal/pkg/ingestion"
	"github.com/numary/payments/internal/pkg/integration"
	"github.com/numary/payments/internal/pkg/payments"
	"github.com/numary/payments/internal/pkg/task"
	"github.com/numary/payments/internal/pkg/writeonly"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/go-libs/sharedpublish"
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
	useScopes := viper.GetBool(authBearerUseScopesFlag)

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

					err := container.Provide(func() writeonly.Storage {
						return writeonly.NewMongoDBStorage(db, loader.Name(), descriptor)
					})
					if err != nil {
						panic(err)
					}

					return container, nil
				}), resolver, maxTasks)
			})

			return integration.NewConnectorManager[ConnectorConfig, TaskDescriptor](logger, connectorStore, loader, schedulerFactory)
		}),
		fx.Provide(fx.Annotate(func(cm *integration.ConnectorManager[ConnectorConfig, TaskDescriptor]) connectorHandler {
			return connectorHandler{
				Handler: connectorRouter(loader.Name(), useScopes, cm),
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

func connectorRouter[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](
	name string,
	useScopes bool,
	manager *integration.ConnectorManager[Config, Descriptor],
) *mux.Router {
	r := mux.NewRouter()

	r.Path("/" + name).Methods(http.MethodPost).Handler(
		wrapHandler(useScopes, install(manager), scopeWriteConnectors),
	)

	r.Path("/" + name + "/reset").Methods(http.MethodPost).Handler(
		wrapHandler(useScopes, reset(manager), scopeWriteConnectors),
	)

	r.Path("/" + name).Methods(http.MethodDelete).Handler(
		wrapHandler(useScopes, uninstall(manager), scopeWriteConnectors),
	)

	r.Path("/" + name + "/config").Methods(http.MethodGet).Handler(
		wrapHandler(useScopes, readConfig(manager), scopeReadConnectors, scopeWriteConnectors),
	)

	r.Path("/" + name + "/tasks").Methods(http.MethodGet).Handler(
		wrapHandler(useScopes, listTasks(manager), scopeReadConnectors, scopeWriteConnectors),
	)

	r.Path("/" + name + "/tasks/{taskId}").Methods(http.MethodGet).Handler(
		wrapHandler(useScopes, readTask(manager), scopeReadConnectors, scopeWriteConnectors),
	)

	return r
}
