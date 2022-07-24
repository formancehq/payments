package cmd

import (
	"context"
	"net"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bombsimon/logrusr/v3"
	"github.com/gorilla/mux"
	"github.com/numary/go-libs/oauth2/oauth2introspect"
	"github.com/numary/go-libs/sharedauth"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/go-libs/sharedlogging/sharedlogginglogrus"
	"github.com/numary/go-libs/sharedotlp"
	"github.com/numary/go-libs/sharedotlp/sharedotlptraces"
	"github.com/numary/go-libs/sharedpublish"
	"github.com/numary/go-libs/sharedpublish/sharedpublishhttp"
	"github.com/numary/go-libs/sharedpublish/sharedpublishkafka"
	"github.com/numary/payments/pkg/api"
	"github.com/numary/payments/pkg/bridge/cdi"
	"github.com/numary/payments/pkg/bridge/connectors/stripe"
	"github.com/numary/payments/pkg/bridge/connectors/wise"
	bridgeHttp "github.com/numary/payments/pkg/bridge/http"
	"github.com/numary/payments/pkg/database"
	paymentapi "github.com/numary/payments/pkg/http"
	"github.com/pkg/errors"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"github.com/xdg-go/scram"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.uber.org/fx"
)

const (
	mongodbUriFlag                       = "mongodb-uri"
	mongodbDatabaseFlag                  = "mongodb-database"
	otelTracesFlag                       = "otel-traces"
	otelTracesExporterFlag               = "otel-traces-exporter"
	otelTracesExporterJaegerEndpointFlag = "otel-traces-exporter-jaeger-endpoint"
	otelTracesExporterJaegerUserFlag     = "otel-traces-exporter-jaeger-user"
	otelTracesExporterJaegerPasswordFlag = "otel-traces-exporter-jaeger-password"
	otelTracesExporterOTLPModeFlag       = "otel-traces-exporter-otlp-mode"
	otelTracesExporterOTLPEndpointFlag   = "otel-traces-exporter-otlp-endpoint"
	otelTracesExporterOTLPInsecureFlag   = "otel-traces-exporter-otlp-insecure"
	envFlag                              = "env"
	publisherKafkaEnabledFlag            = "publisher-kafka-enabled"
	publisherKafkaBrokerFlag             = "publisher-kafka-broker"
	publisherKafkaSASLEnabled            = "publisher-kafka-sasl-enabled"
	publisherKafkaSASLUsername           = "publisher-kafka-sasl-username"
	publisherKafkaSASLPassword           = "publisher-kafka-sasl-password"
	publisherKafkaSASLMechanism          = "publisher-kafka-sasl-mechanism"
	publisherKafkaSASLScramSHASize       = "publisher-kafka-sasl-scram-sha-size"
	publisherKafkaTLSEnabled             = "publisher-kafka-tls-enabled"
	publisherTopicMappingFlag            = "publisher-topic-mapping"
	publisherHttpEnabledFlag             = "publisher-http-enabled"
	authBasicEnabledFlag                 = "auth-basic-enabled"
	authBasicCredentialsFlag             = "auth-basic-credentials"
	authBearerEnabledFlag                = "auth-bearer-enabled"
	authBearerIntrospectUrlFlag          = "auth-bearer-introspect-url"
	authBearerAudienceFlag               = "auth-bearer-audience"
	authBearerAudiencesWildcardFlag      = "auth-bearer-audiences-wildcard"
	authBearerUseScopesFlag              = "auth-bearer-use-scopes"

	serviceName = "Payments"
)

func NewServer() *cobra.Command {
	return &cobra.Command{
		Use:          "server",
		Short:        "Launch server",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			l := logrus.New()
			if viper.GetBool(debugFlag) {
				l.SetLevel(logrus.DebugLevel)
			}
			if viper.GetBool(otelTracesFlag) {
				l.AddHook(otellogrus.NewHook(otellogrus.WithLevels(
					logrus.PanicLevel,
					logrus.FatalLevel,
					logrus.ErrorLevel,
					logrus.WarnLevel,
				)))
				l.SetFormatter(&logrus.JSONFormatter{})
			}
			sharedlogging.SetFactory(sharedlogging.StaticLoggerFactory(sharedlogginglogrus.New(l)))

			// Add a dedicated logger for opentelemetry in case of error
			otel.SetLogger(logrusr.New(logrus.New().WithField("component", "otlp")))

			mongodbUri := viper.GetString(mongodbUriFlag)
			if mongodbUri == "" {
				return errors.New("missing mongodb uri")
			}

			mongodbDatabase := viper.GetString(mongodbDatabaseFlag)
			if mongodbDatabase == "" {
				return errors.New("missing mongodb database name")
			}

			topics := viper.GetStringSlice(publisherTopicMappingFlag)
			mapping := make(map[string]string)
			for _, topic := range topics {
				parts := strings.SplitN(topic, ":", 2)
				if len(parts) != 2 {
					panic("invalid topic flag")
				}
				mapping[parts[0]] = parts[1]
			}

			options := make([]fx.Option, 0)
			if !viper.GetBool(debugFlag) {
				options = append(options, fx.NopLogger)
			}
			//if viper.GetBool(otelTracesFlag) {
			//	options = append(options, database.MongoMonitor())
			//}
			options = append(options,
				database.MongoModule(mongodbUri, mongodbDatabase),
				sharedotlptraces.TracesModule(sharedotlptraces.ModuleConfig{
					Exporter: viper.GetString(otelTracesExporterFlag),
					OTLPConfig: &sharedotlptraces.OTLPConfig{
						Mode:     viper.GetString(otelTracesExporterOTLPModeFlag),
						Endpoint: viper.GetString(otelTracesExporterOTLPEndpointFlag),
						Insecure: viper.GetBool(otelTracesExporterOTLPInsecureFlag),
					},
				}),
				fx.Provide(fx.Annotate(func(p message.Publisher) *sharedpublish.TopicMapperPublisher {
					return sharedpublish.NewTopicMapperPublisher(p, mapping)
				}, fx.As(new(sharedpublish.Publisher)))),
				HTTPModule(),
			)

			options = append(options, sharedpublish.Module())
			switch {
			case viper.GetBool(publisherHttpEnabledFlag):
				options = append(options, sharedpublishhttp.Module())
			case viper.GetBool(publisherKafkaEnabledFlag):
				options = append(options,
					sharedpublishkafka.Module(serviceName, viper.GetStringSlice(publisherKafkaBrokerFlag)...),
					sharedpublishkafka.ProvideSaramaOption(
						sharedpublishkafka.WithConsumerReturnErrors(),
						sharedpublishkafka.WithProducerReturnSuccess(),
					),
				)
				if viper.GetBool(publisherKafkaTLSEnabled) {
					options = append(options, sharedpublishkafka.ProvideSaramaOption(sharedpublishkafka.WithTLS()))
				}
				if viper.GetBool(publisherKafkaSASLEnabled) {
					options = append(options, sharedpublishkafka.ProvideSaramaOption(
						sharedpublishkafka.WithSASLEnabled(),
						sharedpublishkafka.WithSASLCredentials(
							viper.GetString(publisherKafkaSASLUsername),
							viper.GetString(publisherKafkaSASLPassword),
						),
						sharedpublishkafka.WithSASLMechanism(sarama.SASLMechanism(viper.GetString(publisherKafkaSASLMechanism))),
						sharedpublishkafka.WithSASLScramClient(func() sarama.SCRAMClient {
							var fn scram.HashGeneratorFcn
							switch viper.GetInt(publisherKafkaSASLScramSHASize) {
							case 512:
								fn = sharedpublishkafka.SHA512
							case 256:
								fn = sharedpublishkafka.SHA256
							default:
								panic("sha size not handled")
							}
							return &sharedpublishkafka.XDGSCRAMClient{
								HashGeneratorFcn: fn,
							}
						}),
					))
				}
			}

			err := fx.New(options...).Start(cmd.Context())
			if err != nil {
				return err
			}
			<-cmd.Context().Done()
			return nil
		}}
}

func HTTPModule() fx.Option {
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
		fx.Provide(fx.Annotate(func(db *mongo.Database, client *mongo.Client, handlers []cdi.ConnectorHandler) (*mux.Router, error) {

			rootMux := mux.NewRouter()
			if viper.GetBool(otelTracesFlag) {
				rootMux.Use(otelmux.Middleware(serviceName))
			}
			rootMux.Use(
				paymentapi.Recovery(func(ctx context.Context, e interface{}) {
					if viper.GetBool(otelTracesFlag) {
						sharedotlp.RecordAsError(ctx, e)
					} else {
						logrus.Errorln(e)
						debug.PrintStack()
					}
				}),
				cors.New(cors.Options{
					AllowedOrigins:   []string{"*"},
					AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut},
					AllowCredentials: true,
				}).Handler,
			)
			rootMux.Use(func(handler http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					handler.ServeHTTP(w, r)
				})
			})
			rootMux.Path("/_health").Handler(paymentapi.HealthHandler(client))
			rootMux.Path("/_live").Handler(paymentapi.LiveHandler())
			authenticatedRouter := rootMux.Name("authenticated").Subrouter()
			methods := make([]sharedauth.Method, 0)
			if viper.GetBool(authBasicEnabledFlag) {
				credentials := sharedauth.Credentials{}
				for _, kv := range viper.GetStringSlice(authBasicCredentialsFlag) {
					parts := strings.SplitN(kv, ":", 2)
					credentials[parts[0]] = sharedauth.Credential{
						Password: parts[1],
						Scopes:   append(api.AllScopes, bridgeHttp.AllScopes...),
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
			if len(methods) > 0 {
				authenticatedRouter.Use(sharedauth.Middleware(methods...))
			}
			connectorsSubRouter := authenticatedRouter.PathPrefix("/connectors").Subrouter()
			for _, h := range handlers {
				connectorsSubRouter.PathPrefix("/" + h.Name).Handler(
					http.StripPrefix("/connectors", h.Handler),
				)
			}
			authenticatedRouter.PathPrefix("/").Handler(
				api.PaymentsRouter(db, viper.GetBool(authBearerUseScopesFlag)),
			)

			return rootMux, nil
		}, fx.ParamTags(``, ``, `group:"connectorHandlers"`))),
		cdi.ConnectorModule[stripe.Config, stripe.TaskDescriptor](
			viper.GetBool(authBearerUseScopesFlag),
			stripe.NewLoader(),
		),
		cdi.ConnectorModule(
			viper.GetBool(authBearerUseScopesFlag),
			wise.NewLoader(),
		),
	)
}
