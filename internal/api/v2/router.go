package v2

import (
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/auth"
	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/go-chi/chi/v5"
)

func newRouter(backend backend.Backend, info api.ServiceInfo, a auth.Authenticator, debug bool) *chi.Mux {
	r := chi.NewRouter()

	r.Get("/_info", api.InfoHandler(info))

	r.Group(func(r chi.Router) {
		r.Use(service.OTLPMiddleware("payments", debug))

		// Public routes
		r.Group(func(r chi.Router) {
			r.Post("/connectors/webhooks/{connector}/connectorID", connectorsWebhooks(backend))
		})

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(a))

			// Accounts
			r.Route("/accounts", func(r chi.Router) {
				r.Get("/", accountsList(backend))
				r.Post("/", accountsCreate(backend))

				r.Route("/{accountID}", func(r chi.Router) {
					r.Get("/", accountsGet(backend))
					r.Get("/balances", accountsBalances(backend))
				})
			})

			// Bank Accounts
			r.Route("/bank-accounts", func(r chi.Router) {
				r.Post("/", bankAccountsCreate(backend))
				r.Get("/", bankAccountsList(backend))

				r.Route("/{bankAccountID}", func(r chi.Router) {
					r.Get("/", bankAccountsGet(backend))
					r.Patch("/metadata", bankAccountsUpdateMetadata(backend))
					r.Post("/forward", bankAccountsForwardToConnector(backend))
				})
			})

			// Payments
			r.Route("/payments", func(r chi.Router) {
				r.Get("/", paymentsList(backend))
				r.Post("/", paymentsCreate(backend))

				r.Route("/{paymentID}", func(r chi.Router) {
					r.Get("/", paymentsGet(backend))
					r.Patch("/metadata", paymentsUpdateMetadata(backend))
				})
			})

			// Pools
			r.Route("/pools", func(r chi.Router) {
				r.Post("/", poolsCreate(backend))
				r.Get("/", poolsList(backend))

				r.Route("/{poolID}", func(r chi.Router) {
					r.Get("/", poolsGet(backend))
					r.Delete("/", poolsDelete(backend))
					r.Get("/balances", poolsBalancesAt(backend))

					r.Route("/accounts", func(r chi.Router) {
						r.Post("/", poolsAddAccount(backend))
						r.Delete("/{accountID}", poolsRemoveAccount(backend))
					})
				})
			})

			// Connectors
			r.Route("/connectors", func(r chi.Router) {
				r.Get("/", connectorsList(backend))
				r.Get("/configs", connectorsConfigs(backend))

				r.Route("/{connector}", func(r chi.Router) {
					r.Post("/", connectorsInstall(backend))
					connectorsRouter(backend, r)
				})
			})

			// Transfer Initiations
			r.Route("/transfer-initiations", func(r chi.Router) {
				r.Post("/", transferInitiationsCreate(backend))
				r.Get("/", transferInitiationsList(backend))

				r.Route("/{transferInitiationID}", func(r chi.Router) {
					r.Get("/", transferInitiationsGet(backend))
					r.Delete("/", transferInitiationsDelete(backend))

					r.Post("/status", transferInitiationsUpdateStatus(backend))
					r.Post("/retry", transferInitiationsRetry(backend))
					r.Post("/reverse", transferInitiationsReverse(backend))
				})
			})
		})
	})

	return r
}

func connectorsRouter(backend backend.Backend, r chi.Router) {
	r.Route("/{connectorID}", func(r chi.Router) {
		r.Delete("/", connectorsUninstall(backend))
		r.Get("/config", connectorsConfig(backend))
		r.Post("/reset", connectorsReset(backend))
		r.Get("/tasks", tasksList(backend))
		r.Get("/tasks/{taskID}", tasksGet(backend))
		// TODO(polo): add update config handler
	})
}

func connectorID(r *http.Request) string {
	return chi.URLParam(r, "connectorID")
}

func connectorProvider(r *http.Request) string {
	return chi.URLParam(r, "connector")
}

func accountID(r *http.Request) string {
	return chi.URLParam(r, "accountID")
}

func paymentID(r *http.Request) string {
	return chi.URLParam(r, "paymentID")
}

func poolID(r *http.Request) string {
	return chi.URLParam(r, "poolID")
}

func bankAccountID(r *http.Request) string {
	return chi.URLParam(r, "bankAccountID")
}

func taskID(r *http.Request) string {
	return chi.URLParam(r, "taskID")
}

func transferInitiationID(r *http.Request) string {
	return chi.URLParam(r, "transferInitiationID")
}
