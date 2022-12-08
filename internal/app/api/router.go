package api

import (
	"net/http"

	"github.com/formancehq/payments/internal/app/storage"

	"github.com/formancehq/payments/internal/app/integration"
	"github.com/formancehq/payments/internal/app/payments"

	"github.com/formancehq/go-libs/sharedauth"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

func httpRouter(store *storage.Storage, connectorHandlers []connectorHandler) (*mux.Router, error) {
	rootMux := mux.NewRouter()

	if viper.GetBool(otelTracesFlag) {
		rootMux.Use(otelmux.Middleware(serviceName))
	}

	rootMux.Use(recoveryHandler(httpRecoveryFunc))
	rootMux.Use(httpCorsHandler())
	rootMux.Use(httpServeFunc)

	rootMux.Path("/_health").Handler(healthHandler(store))
	rootMux.Path("/_live").Handler(liveHandler())

	authGroup := rootMux.Name("authenticated").Subrouter()

	if methods := sharedAuthMethods(); len(methods) > 0 {
		authGroup.Use(sharedauth.Middleware(methods...))
	}

	authGroup.HandleFunc("/connectors", readConnectorsHandler(store))
	connectorGroup := authGroup.PathPrefix("/connectors").Subrouter()

	connectorGroup.Path("/configs").Handler(connectorConfigsHandler())

	for _, h := range connectorHandlers {
		connectorGroup.PathPrefix("/" + h.Name).Handler(
			http.StripPrefix("/connectors", h.Handler),
		)
	}

	authGroup.Path("/payments").Methods(http.MethodGet).Handler(listPaymentsHandler(store))
	authGroup.Path("/payments/{paymentID}").Methods(http.MethodGet).Handler(readPaymentHandler(store))

	// TODO: It's not ideal to define it explicitly here
	// Refactor it when refactoring the HTTP lib.
	connectorGroup.Path("/stripe/transfers").Methods(http.MethodPost).
		Handler(handleStripeTransfers(store))

	return rootMux, nil
}

func connectorRouter[Config payments.ConnectorConfigObject, Descriptor payments.TaskDescriptor](
	name string,
	manager *integration.ConnectorManager[Config, Descriptor],
) *mux.Router {
	r := mux.NewRouter()

	r.Path("/" + name).Methods(http.MethodPost).Handler(install(manager))

	r.Path("/" + name + "/reset").Methods(http.MethodPost).Handler(reset(manager))

	r.Path("/" + name).Methods(http.MethodDelete).Handler(uninstall(manager))

	r.Path("/" + name + "/config").Methods(http.MethodGet).Handler(readConfig(manager))

	r.Path("/" + name + "/tasks").Methods(http.MethodGet).Handler(listTasks(manager))

	r.Path("/" + name + "/tasks/{taskID}").Methods(http.MethodGet).Handler(readTask(manager))

	return r
}
