package api

import (
	"context"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/numary/payments/internal/pkg/connectors/bankingcircle"
	"github.com/numary/payments/internal/pkg/connectors/currencycloud"

	"github.com/gorilla/mux"
	"github.com/numary/go-libs/oauth2/oauth2introspect"
	"github.com/numary/go-libs/sharedauth"
	sharedotlp "github.com/numary/go-libs/sharedotlp/pkg"
	"github.com/numary/payments/internal/pkg/connectors/dummypay"
	"github.com/numary/payments/internal/pkg/connectors/modulr"
	"github.com/numary/payments/internal/pkg/connectors/stripe"
	"github.com/numary/payments/internal/pkg/connectors/wise"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.uber.org/fx"
)

//nolint:gosec // false positive
const (
	otelTracesFlag                  = "otel-traces"
	authBasicEnabledFlag            = "auth-basic-enabled"
	authBasicCredentialsFlag        = "auth-basic-credentials"
	authBearerEnabledFlag           = "auth-bearer-enabled"
	authBearerIntrospectURLFlag     = "auth-bearer-introspect-url"
	authBearerAudienceFlag          = "auth-bearer-audience"
	authBearerAudiencesWildcardFlag = "auth-bearer-audiences-wildcard"

	serviceName = "Payments"
)

func HTTPModule() fx.Option {
	return fx.Options(
		fx.Invoke(func(m *mux.Router, lc fx.Lifecycle) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						//nolint:gomnd // allow timeout values
						srv := &http.Server{
							Handler:      m,
							Addr:         "0.0.0.0:8080",
							WriteTimeout: 15 * time.Second,
							ReadTimeout:  15 * time.Second,
						}

						err := srv.ListenAndServe()
						if err != nil {
							panic(err)
						}
					}()

					return nil
				},
			})
		}),
		fx.Provide(fx.Annotate(httpRouter, fx.ParamTags(``, ``, `group:"connectorHandlers"`))),
		addConnector[dummypay.Config, dummypay.TaskDescriptor](dummypay.NewLoader()),
		addConnector[modulr.Config, modulr.TaskDescriptor](modulr.NewLoader()),
		addConnector[stripe.Config, stripe.TaskDescriptor](stripe.NewLoader()),
		addConnector[wise.Config, wise.TaskDescriptor](wise.NewLoader()),
		addConnector[currencycloud.Config, currencycloud.TaskDescriptor](currencycloud.NewLoader()),
		addConnector[bankingcircle.Config, bankingcircle.TaskDescriptor](bankingcircle.NewLoader()),
	)
}

func httpRouter(db *mongo.Database, client *mongo.Client, handlers []connectorHandler) (*mux.Router, error) {
	rootMux := mux.NewRouter()

	if viper.GetBool(otelTracesFlag) {
		rootMux.Use(otelmux.Middleware(serviceName))
	}

	rootMux.Use(recoveryHandler(httpRecoveryFunc))
	rootMux.Use(httpCorsHandler())
	rootMux.Use(httpServeFunc)

	rootMux.Path("/_health").Handler(healthHandler(client))
	rootMux.Path("/_live").Handler(liveHandler())

	authGroup := rootMux.Name("authenticated").Subrouter()

	if methods := sharedAuthMethods(); len(methods) > 0 {
		authGroup.Use(sharedauth.Middleware(methods...))
	}

	connectorGroup := authGroup.PathPrefix("/connectors").Subrouter()

	for _, h := range handlers {
		connectorGroup.PathPrefix("/" + h.Name).Handler(
			http.StripPrefix("/connectors", h.Handler),
		)
	}

	authGroup.PathPrefix("/").Handler(paymentsRouter(db))

	return rootMux, nil
}

func httpRecoveryFunc(ctx context.Context, e interface{}) {
	if viper.GetBool(otelTracesFlag) {
		sharedotlp.RecordAsError(ctx, e)
	} else {
		logrus.Errorln(e)
		debug.PrintStack()
	}
}

func httpCorsHandler() func(http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut},
		AllowCredentials: true,
	}).Handler
}

func httpServeFunc(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		handler.ServeHTTP(w, r)
	})
}

func sharedAuthMethods() []sharedauth.Method {
	methods := make([]sharedauth.Method, 0)

	if viper.GetBool(authBasicEnabledFlag) {
		credentials := sharedauth.Credentials{}

		for _, kv := range viper.GetStringSlice(authBasicCredentialsFlag) {
			parts := strings.SplitN(kv, ":", 2)
			credentials[parts[0]] = sharedauth.Credential{
				Password: parts[1],
			}
		}

		methods = append(methods, sharedauth.NewHTTPBasicMethod(credentials))
	}

	if viper.GetBool(authBearerEnabledFlag) {
		methods = append(methods, sharedauth.NewHttpBearerMethod(
			sharedauth.NewIntrospectionValidator(
				oauth2introspect.NewIntrospecter(viper.GetString(authBearerIntrospectURLFlag)),
				viper.GetBool(authBearerAudiencesWildcardFlag),
				sharedauth.AudienceIn(viper.GetStringSlice(authBearerAudienceFlag)...),
			),
		))
	}

	return methods
}
