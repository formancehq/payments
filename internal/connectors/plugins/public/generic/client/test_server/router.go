package testserver

import (
	"net/http"

	"github.com/formancehq/go-libs/v2/health"
	"github.com/go-chi/chi/v5"
)

func newRouter(a *API, healthController *health.HealthController) *chi.Mux {
	r := chi.NewRouter()

	r.Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			handler.ServeHTTP(w, r)
		})
	})

	r.Get("/_healthcheck", healthController.Check)

	r.Get("/accounts", a.accountsList())
	r.Get("/accounts/{accountID}/balances", a.balancesList())
	r.Get("/beneficiaries", a.beneficiariesList())
	r.Get("/transactions", a.transactionsList())

	return r
}
