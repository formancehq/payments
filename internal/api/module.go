package api

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/formancehq/go-libs/v5/pkg/transport/api"
	"github.com/formancehq/go-libs/v5/pkg/authn/jwt"
	"github.com/formancehq/go-libs/v5/pkg/service/health"
	"github.com/formancehq/go-libs/v5/pkg/transport/httpserver"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/services"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
)

func TagVersion() fx.Annotation {
	return fx.ResultTags(`group:"apiVersions"`)
}

func NewModule(bind string, debug bool) fx.Option {
	return fx.Options(
		fx.Invoke(fx.Annotate(func(m *chi.Mux, lc fx.Lifecycle) {
			hook := httpserver.NewHook(m, httpserver.WithAddress(bind))
			lc.Append(fx.Hook{OnStart: hook.OnStart, OnStop: hook.OnStop})
		}, fx.ParamTags(`name:"apiRouter"`, ``))),
		fx.Provide(fx.Annotate(func(
			backend backend.Backend,
			info api.ServiceInfo,
			healthController *health.HealthController,
			a jwt.Authenticator,
			publisher message.Publisher,
			versions ...Version,
		) *chi.Mux {
			return NewRouter(backend, info, healthController, a, publisher, debug, versions...)
		}, fx.ParamTags(``, ``, ``, ``, ``, `group:"apiVersions"`), fx.ResultTags(`name:"apiRouter"`))),
		fx.Provide(func(storage storage.Storage, engine engine.Engine) backend.Backend {
			return services.New(storage, engine, debug)
		}),
	)
}
