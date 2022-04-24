package cmd

import (
	"context"
	"github.com/Shopify/sarama"
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
	payment "github.com/numary/payments/pkg"
	"github.com/pkg/errors"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"github.com/xdg-go/scram"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"go.uber.org/fx"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"
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
	debugFlag                            = "debug"
	envFlag                              = "env"
	esIndexFlag                          = "es-index"
	esAddressFlag                        = "es-address"
	esInsecureFlag                       = "es-insecure"
	esUsernameFlag                       = "es-username"
	esPasswordFlag                       = "es-password"
	noAuthFlag                           = "no-auth"
	httpBasicFlag                        = "http-basic"
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

	serviceName = "Payments"
)

var (
	Version = "latest"
)

func MongoModule(uri string, dbName string) fx.Option {
	return fx.Options(
		fx.Supply(options.Client().ApplyURI(uri)),
		fx.Provide(func(opts *options.ClientOptions) (*mongo.Client, error) {
			return mongo.NewClient(opts)
		}),
		fx.Provide(func(client *mongo.Client) *mongo.Database {
			return client.Database(dbName)
		}),
		fx.Invoke(func(lc fx.Lifecycle, client *mongo.Client) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					err := client.Connect(context.Background())
					if err != nil {
						return err
					}
					sharedlogging.Debug("Ping database...")
					ctx, _ = context.WithDeadline(ctx, time.Now().Add(time.Second*5))
					err = client.Ping(ctx, readpref.Primary())
					if err != nil {
						return err
					}
					return nil
				},
			})
		}),
	)
}

func MongoMonitor() fx.Option {
	return fx.Decorate(func(opts *options.ClientOptions) *options.ClientOptions {
		opts.SetMonitor(otelmongo.NewMonitor())
		return opts
	})
}

func ServiceModule() fx.Option {
	return fx.Provide(fx.Annotate(
		payment.NewDefaultService, fx.ParamTags(``, `group:"serviceOptions"`), fx.As(new(payment.Service)),
	))
}

func ServicePublishModule() fx.Option {
	return fx.Provide(fx.Annotate(func(p *sharedpublish.TopicMapperPublisher) payment.ServiceOption {
		return payment.WithPublisher(p)
	}, fx.ResultTags(`group:"serviceOptions"`)))
}

func HTTPModule() fx.Option {
	return fx.Options(
		fx.Invoke(func(m *mux.Router, lc fx.Lifecycle) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go http.ListenAndServe(":8080", m)
					return nil
				},
			})
		}),
		fx.Provide(func(srv payment.Service, client *mongo.Client) (*mux.Router, error) {

			m := payment.NewMux(srv)
			if viper.GetBool(otelTracesFlag) {
				m.Use(otelmux.Middleware(serviceName))
			}
			methods := make([]sharedauth.Method, 0)
			if viper.GetBool(authBasicEnabledFlag) {
				credentials := make(map[string]string)
				for _, kv := range viper.GetStringSlice(authBasicCredentialsFlag) {
					parts := strings.SplitN(kv, ":", 2)
					credentials[parts[0]] = parts[1]
				}
				methods = append(methods, sharedauth.NewHTTPBasicMethod(credentials))
			}
			if viper.GetBool(authBearerEnabledFlag) {
				methods = append(methods, sharedauth.NewHttpBearerMethod(
					oauth2introspect.NewIntrospecter(viper.GetString(authBearerIntrospectUrlFlag)),
					viper.GetBool(authBearerAudiencesWildcardFlag),
					viper.GetStringSlice(authBearerAudienceFlag)...,
				))
			}
			if len(methods) > 0 {
				m.Use(sharedauth.Middleware(methods...))
			}

			rootMux := mux.NewRouter()
			rootMux.Use(
				payment.Recovery(func(ctx context.Context, e interface{}) {
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
			rootMux.Path("/_health").Handler(payment.HealthHandler(client))
			rootMux.Path("/_live").Handler(payment.LiveHandler())
			rootMux.PathPrefix("/").Handler(m)

			return rootMux, nil
		}),
	)
}

var rootCmd = &cobra.Command{
	Use:          "payment",
	Short:        "Payment api",
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
		options = append(options,
			MongoModule(mongodbUri, mongodbDatabase),
			MongoMonitor(),
			sharedotlptraces.TracesModule(sharedotlptraces.ModuleConfig{
				Exporter: viper.GetString(otelTracesExporterFlag),
				OTLPConfig: &sharedotlptraces.OTLPConfig{
					Mode:     viper.GetString(otelTracesExporterOTLPModeFlag),
					Endpoint: viper.GetString(otelTracesExporterOTLPEndpointFlag),
					Insecure: viper.GetBool(otelTracesExporterOTLPInsecureFlag),
				},
			}),
			sharedpublish.TopicMapperPublisherModule(mapping),
			ServiceModule(),
			ServicePublishModule(),
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
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.Flags().Bool(debugFlag, false, "Debug mode")
	rootCmd.Flags().String(mongodbUriFlag, "mongodb://localhost:27017", "MongoDB address")
	rootCmd.Flags().String(mongodbDatabaseFlag, "payments", "MongoDB database name")
	rootCmd.Flags().Bool(otelTracesFlag, false, "Enable OpenTelemetry traces support")
	rootCmd.Flags().String(otelTracesExporterFlag, "stdout", "OpenTelemetry traces exporter")
	rootCmd.Flags().String(otelTracesExporterJaegerEndpointFlag, "", "OpenTelemetry traces Jaeger exporter endpoint")
	rootCmd.Flags().String(otelTracesExporterJaegerUserFlag, "", "OpenTelemetry traces Jaeger exporter user")
	rootCmd.Flags().String(otelTracesExporterJaegerPasswordFlag, "", "OpenTelemetry traces Jaeger exporter password")
	rootCmd.Flags().String(otelTracesExporterOTLPModeFlag, "grpc", "OpenTelemetry traces OTLP exporter mode (grpc|http)")
	rootCmd.Flags().String(otelTracesExporterOTLPEndpointFlag, "", "OpenTelemetry traces grpc endpoint")
	rootCmd.Flags().Bool(otelTracesExporterOTLPInsecureFlag, false, "OpenTelemetry traces grpc insecure")
	rootCmd.Flags().String(esIndexFlag, "ledger", "Index on which push new payments")
	rootCmd.Flags().StringSlice(esAddressFlag, []string{}, "ES addresses")
	rootCmd.Flags().Bool(esInsecureFlag, false, "Insecure es connection (no valid tls certificate)")
	rootCmd.Flags().String(esUsernameFlag, "", "ES username")
	rootCmd.Flags().String(esPasswordFlag, "", "ES password")
	rootCmd.Flags().String(envFlag, "local", "Environment")
	rootCmd.Flags().Bool(noAuthFlag, false, "Disable authentication")
	rootCmd.Flags().String(httpBasicFlag, "", "HTTP basic authentication")
	rootCmd.Flags().Bool(publisherKafkaEnabledFlag, false, "Publish write events to kafka")
	rootCmd.Flags().StringSlice(publisherKafkaBrokerFlag, []string{}, "Kafka address is kafka enabled")
	rootCmd.Flags().StringSlice(publisherTopicMappingFlag, []string{}, "Define mapping between internal event types and topics")
	rootCmd.Flags().Bool(publisherHttpEnabledFlag, false, "Sent write event to http endpoint")
	rootCmd.Flags().Bool(publisherKafkaSASLEnabled, false, "Enable SASL authentication on kafka publisher")
	rootCmd.Flags().String(publisherKafkaSASLUsername, "", "SASL username")
	rootCmd.Flags().String(publisherKafkaSASLPassword, "", "SASL password")
	rootCmd.Flags().String(publisherKafkaSASLMechanism, "", "SASL authentication mechanism")
	rootCmd.Flags().Int(publisherKafkaSASLScramSHASize, 512, "SASL SCRAM SHA size")
	rootCmd.Flags().Bool(publisherKafkaTLSEnabled, false, "Enable TLS to connect on kafka")
	rootCmd.Flags().Bool(authBasicEnabledFlag, false, "Enable basic auth")
	rootCmd.Flags().StringSlice(authBasicCredentialsFlag, []string{}, "HTTP basic auth credentials (<username>:<password>)")
	rootCmd.Flags().Bool(authBearerEnabledFlag, false, "Enable bearer auth")
	rootCmd.Flags().String(authBearerIntrospectUrlFlag, "", "OAuth2 introspect URL")
	rootCmd.Flags().StringSlice(authBearerAudienceFlag, []string{}, "Allowed audiences")
	rootCmd.Flags().Bool(authBearerAudiencesWildcardFlag, false, "Don't check audience")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	viper.BindPFlags(rootCmd.Flags())
}
