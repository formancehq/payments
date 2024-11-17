package testserver

import (
	"time"

	"github.com/formancehq/go-libs/v2/health"
	"github.com/formancehq/go-libs/v2/httpserver"
	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
)

func NewModule(
	nbAccounts int,
	bind string,
) fx.Option {
	return fx.Options(
		fx.Supply(&API{
			firstTimeCreation: time.Now().UTC(),
			nbAccounts:        nbAccounts,
		}),
		fx.Invoke(func(m *chi.Mux, lc fx.Lifecycle) {
			lc.Append(httpserver.NewHook(m, httpserver.WithAddress(bind)))
		}),
		fx.Provide(func(a *API, healthController *health.HealthController) *chi.Mux {
			return newRouter(a, healthController)
		}),
	)
}
