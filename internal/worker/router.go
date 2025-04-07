package worker

import (
	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/go-libs/v3/health"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter returns minimal router containing health checks
func NewRouter(
	info api.ServiceInfo,
	healthController *health.HealthController,
) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Get("/_healthcheck", healthController.Check)
	r.Get("/_info", api.InfoHandler(info))
	return r
}
