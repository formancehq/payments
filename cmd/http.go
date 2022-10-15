package cmd

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/numary/go-libs/oauth2/oauth2introspect"
	"github.com/numary/go-libs/sharedauth"
	sharedotlp "github.com/numary/go-libs/sharedotlp/pkg"
	"github.com/numary/payments/internal/app/api"
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
	"net"
	"net/http"
	"runtime/debug"
	"strings"
)

func httpModule() fx.Option {
	return fx.Options(
		fx.Invoke(func(m *mux.Router, lc fx.Lifecycle) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					conn, err := net.Listen("tcp", ":8080")
					if err != nil {
						return err
					}

					go func() {
						err := http.Serve(conn, m)
						if err != nil {
							panic(err)
						}
					}()

					return nil
				},
			})
		}),
		fx.Provide(fx.Annotate(httpRouter), fx.ParamTags(``, ``, `group:"connectorHandlers"`)),
		api.ConnectorModule[stripe.Config, stripe.TaskDescriptor](
			viper.GetBool(authBearerUseScopesFlag),
			stripe.NewLoader(),
		),
		api.ConnectorModule[dummypay.Config, dummypay.TaskDescriptor](
			viper.GetBool(authBearerUseScopesFlag),
			dummypay.NewLoader(),
		),
		api.ConnectorModule(
			viper.GetBool(authBearerUseScopesFlag),
			wise.NewLoader(),
		),
		api.ConnectorModule(
			viper.GetBool(authBearerUseScopesFlag),
			modulr.NewLoader(),
		),
	)
}

func httpRouter(db *mongo.Database, client *mongo.Client, handlers []api.ConnectorHandler) (*mux.Router, error) {
	rootMux := mux.NewRouter()

	if viper.GetBool(otelTracesFlag) {
		rootMux.Use(otelmux.Middleware(serviceName))
	}

	rootMux.Use(api.Recovery(httpRecoveryFunc))
	rootMux.Use(httpCorsHandler())
	rootMux.Use(httpServeFunc)

	rootMux.Path("/_health").Handler(api.HealthHandler(client))
	rootMux.Path("/_live").Handler(api.LiveHandler())

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

	authGroup.PathPrefix("/").Handler(
		api.PaymentsRouter(db, viper.GetBool(authBearerUseScopesFlag)),
	)

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
				Scopes:   api.AllScopes,
			}
		}
		methods = append(methods, sharedauth.NewHTTPBasicMethod(credentials))
	}

	if viper.GetBool(authBearerEnabledFlag) {
		methods = append(methods, sharedauth.NewHttpBearerMethod(
			sharedauth.NewIntrospectionValidator(
				oauth2introspect.NewIntrospecter(viper.GetString(authBearerIntrospectUrlFlag)),
				viper.GetBool(authBearerAudiencesWildcardFlag),
				sharedauth.AudienceIn(viper.GetStringSlice(authBearerAudienceFlag)...),
			),
		))
	}

	return methods
}
