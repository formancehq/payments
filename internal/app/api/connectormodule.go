package api

import (
	"context"
	"github.com/gorilla/mux"
	"net/http"

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

//viper.GetBool(authBearerUseScopesFlag),

//func Connectors(authBearerUseScopesFlag bool) []fx.Option {
//	return []fx.Option{
//		ConnectorModule[stripe.Config, stripe.TaskDescriptor](
//			authBearerUseScopesFlag,
//			stripe.NewLoader(),
//		)}

//ConnectorModule[stripe.Config, stripe.TaskDescriptor](
//	viper.GetBool(authBearerUseScopesFlag),
//	stripe.NewLoader(),
//),
//ConnectorModule[dummypay.Config, dummypay.TaskDescriptor](
//	viper.GetBool(authBearerUseScopesFlag),
//	dummypay.NewLoader(),
//),
//ConnectorModule[modulr.Config, modulr.TaskDescriptor](
//	viper.GetBool(authBearerUseScopesFlag),
//	modulr.NewLoader(),
//),
//ConnectorModule[wise.Config, wise.TaskDescriptor](
//	viper.GetBool(authBearerUseScopesFlag),
//	wise.NewLoader(),
//),
//}

type ConnectorHandler struct {
	Handler http.Handler
	Name    string
}

func ConnectorModule[
	ConnectorConfig payments.ConnectorConfigObject,
	TaskDescriptor payments.TaskDescriptor,
](useScopes bool, loader integration.Loader[ConnectorConfig, TaskDescriptor],
) fx.Option {
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
		fx.Provide(fx.Annotate(func(cm *integration.ConnectorManager[ConnectorConfig, TaskDescriptor]) ConnectorHandler {
			return ConnectorHandler{
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
		wrapHandler(useScopes, Install(manager), scopeWriteConnectors),
	)

	r.Path("/" + name + "/reset").Methods(http.MethodPost).Handler(
		wrapHandler(useScopes, Reset(manager), scopeWriteConnectors),
	)

	r.Path("/" + name).Methods(http.MethodDelete).Handler(
		wrapHandler(useScopes, Uninstall(manager), scopeWriteConnectors),
	)

	r.Path("/" + name + "/config").Methods(http.MethodGet).Handler(
		wrapHandler(useScopes, ReadConfig(manager), scopeReadConnectors, scopeWriteConnectors),
	)

	r.Path("/" + name + "/tasks").Methods(http.MethodGet).Handler(
		wrapHandler(useScopes, ListTasks(manager), scopeReadConnectors, scopeWriteConnectors),
	)

	r.Path("/" + name + "/tasks/{taskId}").Methods(http.MethodGet).Handler(
		wrapHandler(useScopes, ReadTask(manager), scopeReadConnectors, scopeWriteConnectors),
	)

	return r
}
