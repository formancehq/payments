package v3

import (
	"net/http"

	"github.com/formancehq/go-libs/v3/auth"
	"github.com/formancehq/go-libs/v3/service"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/go-chi/chi/v5"
)

func newRouter(backend backend.Backend, a auth.Authenticator, debug bool) *chi.Mux {
	r := chi.NewRouter()
	validator := validation.NewValidator()

	r.Group(func(r chi.Router) {
		r.Use(service.OTLPMiddleware("payments", debug))

		// Public routes
		r.Group(func(r chi.Router) {
			r.Handle("/connectors/webhooks/{connectorID}/*", connectorsWebhooks(backend))
		})

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(a))

			// Accounts
			r.Route("/accounts", func(r chi.Router) {
				r.Get("/", accountsList(backend))
				r.Post("/", accountsCreate(backend, validator))

				r.Route("/{accountID}", func(r chi.Router) {
					r.Get("/", accountsGet(backend))
					r.Get("/balances", accountsBalances(backend))
				})
			})

			// Bank Accounts
			r.Route("/bank-accounts", func(r chi.Router) {
				r.Post("/", bankAccountsCreate(backend, validator))
				r.Get("/", bankAccountsList(backend))

				r.Route("/{bankAccountID}", func(r chi.Router) {
					r.Get("/", bankAccountsGet(backend))
					r.Patch("/metadata", bankAccountsUpdateMetadata(backend))
					r.Post("/forward", bankAccountsForwardToConnector(backend, validator))
				})
			})

			// Payments
			r.Route("/payments", func(r chi.Router) {
				r.Post("/", paymentsCreate(backend, validator))
				r.Get("/", paymentsList(backend))

				r.Route("/{paymentID}", func(r chi.Router) {
					r.Get("/", paymentsGet(backend))
					r.Patch("/metadata", paymentsUpdateMetadata(backend))
				})
			})

			// Pools
			r.Route("/pools", func(r chi.Router) {
				r.Post("/", poolsCreate(backend, validator))
				r.Get("/", poolsList(backend))

				r.Route("/{poolID}", func(r chi.Router) {
					r.Get("/", poolsGet(backend))
					r.Delete("/", poolsDelete(backend))
					r.Get("/balances", poolsBalancesAt(backend))
					r.Get("/balances/latest", poolsBalancesLatest(backend))

					r.Route("/accounts/{accountID}", func(r chi.Router) {
						r.Post("/", poolsAddAccount(backend))
						r.Delete("/", poolsRemoveAccount(backend))
					})
				})
			})

			// Connectors
			r.Route("/connectors", func(r chi.Router) {
				r.Get("/", connectorsList(backend))
				r.Post("/install/{connector}", connectorsInstall(backend))

				r.Get("/configs", connectorsConfigs(backend))

				r.Route("/{connectorID}", func(r chi.Router) {
					r.Delete("/", connectorsUninstall(backend))
					r.Get("/config", connectorsConfig(backend))
					r.Patch("/config", connectorsConfigUpdate(backend))
					r.Post("/reset", connectorsReset(backend))

					r.Get("/schedules", schedulesList(backend))
					r.Route("/schedules/{scheduleID}", func(r chi.Router) {
						r.Get("/", schedulesGet(backend))
						r.Get("/instances", workflowsInstancesList(backend))
					})
				})
			})

			// Tasks
			r.Route("/tasks", func(r chi.Router) {
				r.Route("/{taskID}", func(r chi.Router) {
					r.Get("/", tasksGet(backend))
				})
			})

			// Payment Initiations
			r.Route("/payment-initiations", func(r chi.Router) {
				r.Post("/", paymentInitiationsCreate(backend, validator))
				r.Get("/", paymentInitiationsList(backend))

				r.Route("/{paymentInitiationID}", func(r chi.Router) {
					r.Delete("/", paymentInitiationsDelete(backend))
					r.Get("/", paymentInitiationsGet(backend))
					r.Post("/retry", paymentInitiationsRetry(backend))
					r.Post("/approve", paymentInitiationsApprove(backend))
					r.Post("/reject", paymentInitiationsReject(backend))
					r.Post("/reverse", paymentInitiationsReverse(backend, validator))

					r.Get("/adjustments", paymentInitiationAdjustmentsList(backend))
					r.Get("/payments", paymentInitiationPaymentsList(backend))
				})
			})

			r.Route("/payment-service-users", func(r chi.Router) {
				r.Post("/", paymentServiceUsersCreate(backend, validator))
				r.Get("/", paymentServiceUsersList(backend))

				r.Route("/{paymentServiceUserID}", func(r chi.Router) {
					r.Get("/", paymentServiceUsersGet(backend))

					r.Route("/bank-accounts/{bankAccountID}", func(r chi.Router) {
						r.Post("/", paymentServiceUsersAddBankAccount(backend))
						r.Post("/forward", paymentServiceUsersForwardBankAccountToConnector(backend, validator))
					})
				})
			})
		})
	})

	return r
}

func connector(r *http.Request) string {
	return chi.URLParam(r, "connector")
}

func connectorID(r *http.Request) string {
	return chi.URLParam(r, "connectorID")
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

func paymentServiceUserID(r *http.Request) string {
	return chi.URLParam(r, "paymentServiceUserID")
}

func scheduleID(r *http.Request) string {
	return chi.URLParam(r, "scheduleID")
}

func taskID(r *http.Request) string {
	return chi.URLParam(r, "taskID")
}

func paymentInitiationID(r *http.Request) string {
	return chi.URLParam(r, "paymentInitiationID")
}
