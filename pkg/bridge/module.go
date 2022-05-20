package bridge

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/go-libs/sharedpublish"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/fx"
	http2 "net/http"
)

type ConnectorHandler struct {
	Handler http2.Handler
	Name    string
}

func ConnectorModule[T ConnectorConfigObject, S ConnectorState, C Connector[T, S]](
	useScopes bool,
	controller Loader[T, S, C],
) fx.Option {
	var connector C
	return fx.Options(
		fx.Provide(func(db *mongo.Database, publisher sharedpublish.Publisher) *ConnectorManager[T, S] {
			return NewConnectorManager[T, S, C](db, controller,
				NewDefaultIngester[T, S, C](db, sharedlogging.GetLogger(context.Background()), publisher),
			)
		}),
		fx.Provide(fx.Annotate(func(cm *ConnectorManager[T, S]) ConnectorHandler {
			return ConnectorHandler{
				Handler: ConnectorRouter(connector.Name(), useScopes, cm),
				Name:    connector.Name(),
			}
		}, fx.ResultTags(`group:"connectorHandlers"`))),
		fx.Invoke(func(lc fx.Lifecycle, cm *ConnectorManager[T, S]) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					_ = cm.Restore(ctx)
					return nil
				},
			})
		}),
	)
}
