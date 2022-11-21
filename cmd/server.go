package cmd

import (
	"strings"

	"github.com/Shopify/sarama"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bombsimon/logrusr/v3"
	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/go-libs/sharedlogging/sharedlogginglogrus"
	"github.com/numary/go-libs/sharedotlp/pkg/sharedotlptraces"
	"github.com/numary/go-libs/sharedpublish"
	"github.com/numary/go-libs/sharedpublish/sharedpublishhttp"
	"github.com/numary/go-libs/sharedpublish/sharedpublishkafka"
	"github.com/numary/payments/internal/app/api"
	"github.com/numary/payments/internal/app/database"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"github.com/xdg-go/scram"
	"go.opentelemetry.io/otel"
	"go.uber.org/fx"
)

//nolint:gosec // false positive
const (
	mongodbURIFlag                       = "mongodb-uri"
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
	publisherHTTPEnabledFlag             = "publisher-http-enabled"
	authBasicEnabledFlag                 = "auth-basic-enabled"
	authBasicCredentialsFlag             = "auth-basic-credentials"
	authBearerEnabledFlag                = "auth-bearer-enabled"
	authBearerIntrospectURLFlag          = "auth-bearer-introspect-url"
	authBearerAudienceFlag               = "auth-bearer-audience"
	authBearerAudiencesWildcardFlag      = "auth-bearer-audiences-wildcard"
	authBearerUseScopesFlag              = "auth-bearer-use-scopes"

	serviceName = "Payments"
)

func newServer() *cobra.Command {
	return &cobra.Command{
		Use:          "server",
		Short:        "Launch server",
		SilenceUsage: true,
		RunE:         runServer,
	}
}

func runServer(cmd *cobra.Command, args []string) error {
	setLogger()

	databaseOptions, err := prepareDatabaseOptions()
	if err != nil {
		return err
	}

	options := make([]fx.Option, 0)

	if !viper.GetBool(debugFlag) {
		options = append(options, fx.NopLogger)
	}

	options = append(options, databaseOptions)

	options = append(options, sharedotlptraces.TracesModule(sharedotlptraces.ModuleConfig{
		Exporter: viper.GetString(otelTracesExporterFlag),
		OTLPConfig: &sharedotlptraces.OTLPConfig{
			Mode:     viper.GetString(otelTracesExporterOTLPModeFlag),
			Endpoint: viper.GetString(otelTracesExporterOTLPEndpointFlag),
			Insecure: viper.GetBool(otelTracesExporterOTLPInsecureFlag),
		},
	}))

	options = append(options,
		fx.Provide(fx.Annotate(func(p message.Publisher) *sharedpublish.TopicMapperPublisher {
			return sharedpublish.NewTopicMapperPublisher(p, topicsMapping())
		}, fx.As(new(sharedpublish.Publisher)))))

	options = append(options, api.HTTPModule())
	options = append(options, sharedpublish.Module())

	switch {
	case viper.GetBool(publisherHTTPEnabledFlag):
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
				sharedpublishkafka.WithSASLScramClient(setSCRAMClient),
			))
		}
	}

	err = fx.New(options...).Start(cmd.Context())
	if err != nil {
		return err
	}

	<-cmd.Context().Done()

	return nil
}

func setLogger() {
	log := logrus.New()

	if viper.GetBool(debugFlag) {
		log.SetLevel(logrus.DebugLevel)
	}

	if viper.GetBool(otelTracesFlag) {
		log.AddHook(otellogrus.NewHook(otellogrus.WithLevels(
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
		)))
		log.SetFormatter(&logrus.JSONFormatter{})
	}

	sharedlogging.SetFactory(sharedlogging.StaticLoggerFactory(sharedlogginglogrus.New(log)))

	// Add a dedicated logger for opentelemetry in case of error
	otel.SetLogger(logrusr.New(logrus.New().WithField("component", "otlp")))
}

func prepareDatabaseOptions() (fx.Option, error) {
	mongodbURI := viper.GetString(mongodbURIFlag)
	if mongodbURI == "" {
		return nil, errors.New("missing mongodb uri")
	}

	mongodbDatabase := viper.GetString(mongodbDatabaseFlag)
	if mongodbDatabase == "" {
		return nil, errors.New("missing mongodb database name")
	}

	return database.MongoModule(mongodbURI, mongodbDatabase), nil
}

func topicsMapping() map[string]string {
	topics := viper.GetStringSlice(publisherTopicMappingFlag)
	mapping := make(map[string]string)

	for _, topic := range topics {
		parts := strings.SplitN(topic, ":", 2)
		if len(parts) != 2 {
			panic("invalid topic flag")
		}

		mapping[parts[0]] = parts[1]
	}

	return mapping
}

func setSCRAMClient() sarama.SCRAMClient {
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
}
