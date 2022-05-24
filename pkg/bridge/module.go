package bridge

import (
	"context"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/go-libs/sharedpublish"
	"github.com/numary/payments/pkg"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/fx"
	http2 "net/http"
)

type ConnectorHandler struct {
	Handler http2.Handler
	Name    string
}

func ConnectorModule[CONFIG payments.ConnectorConfigObject, STATE payments.ConnectorState, CONNECTOR Connector[CONFIG, STATE]](
	useScopes bool,
	controller Loader[CONFIG, STATE, CONNECTOR],
) fx.Option {
	var connector CONNECTOR
	return fx.Options(
		fx.Provide(func(db *mongo.Database, publisher sharedpublish.Publisher) *ConnectorManager[CONFIG, STATE] {
			return NewConnectorManager[CONFIG, STATE, CONNECTOR](db, controller,
				NewDefaultIngester[STATE](connector.Name(), db, sharedlogging.GetLogger(context.Background()), publisher),
			)
		}),
		fx.Provide(fx.Annotate(func(cm *ConnectorManager[CONFIG, STATE]) ConnectorHandler {
			return ConnectorHandler{
				Handler: ConnectorRouter(connector.Name(), useScopes, cm),
				Name:    connector.Name(),
			}
		}, fx.ResultTags(`group:"connectorHandlers"`))),
		fx.Invoke(func(lc fx.Lifecycle, cm *ConnectorManager[CONFIG, STATE]) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					_ = cm.Restore(ctx)
					return nil
				},
			})
		}),
	)
}
