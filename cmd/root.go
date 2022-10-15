package cmd

import (
	"fmt"
	"os"
	"strings"

	_ "github.com/bombsimon/logrusr/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	debugFlag = "debug"
)

var (
	Version   = "develop"
	BuildDate = "-"
	Commit    = "-"
)

func rootCommand() *cobra.Command {
	viper.SetDefault("version", Version)

	root := &cobra.Command{
		Use:               "payments",
		Short:             "payments",
		DisableAutoGenTag: true,
	}

	version := newVersion()
	root.AddCommand(version)

	server := newServer()
	root.AddCommand(server)

	root.PersistentFlags().Bool(debugFlag, false, "Debug mode")
	root.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	root.Flags().Bool(debugFlag, false, "Debug mode")
	root.Flags().String(mongodbUriFlag, "mongodb://localhost:27017", "MongoDB address")
	root.Flags().String(mongodbDatabaseFlag, "payments", "MongoDB database name")
	root.Flags().Bool(otelTracesFlag, false, "Enable OpenTelemetry traces support")
	root.Flags().String(otelTracesExporterFlag, "stdout", "OpenTelemetry traces exporter")
	root.Flags().String(otelTracesExporterJaegerEndpointFlag, "", "OpenTelemetry traces Jaeger exporter endpoint")
	root.Flags().String(otelTracesExporterJaegerUserFlag, "", "OpenTelemetry traces Jaeger exporter user")
	root.Flags().String(otelTracesExporterJaegerPasswordFlag, "", "OpenTelemetry traces Jaeger exporter password")
	root.Flags().String(otelTracesExporterOTLPModeFlag, "grpc", "OpenTelemetry traces OTLP exporter mode (grpc|httphelpers)")
	root.Flags().String(otelTracesExporterOTLPEndpointFlag, "", "OpenTelemetry traces grpc endpoint")
	root.Flags().Bool(otelTracesExporterOTLPInsecureFlag, false, "OpenTelemetry traces grpc insecure")
	root.Flags().String(envFlag, "local", "Environment")
	root.Flags().Bool(publisherKafkaEnabledFlag, false, "Publish write events to kafka")
	root.Flags().StringSlice(publisherKafkaBrokerFlag, []string{}, "Kafka address is kafka enabled")
	root.Flags().StringSlice(publisherTopicMappingFlag, []string{}, "Define mapping between internal event types and topics")
	root.Flags().Bool(publisherHttpEnabledFlag, false, "Sent write event to httphelpers endpoint")
	root.Flags().Bool(publisherKafkaSASLEnabled, false, "Enable SASL authentication on kafka publisher")
	root.Flags().String(publisherKafkaSASLUsername, "", "SASL username")
	root.Flags().String(publisherKafkaSASLPassword, "", "SASL password")
	root.Flags().String(publisherKafkaSASLMechanism, "", "SASL authentication mechanism")
	root.Flags().Int(publisherKafkaSASLScramSHASize, 512, "SASL SCRAM SHA size")
	root.Flags().Bool(publisherKafkaTLSEnabled, false, "Enable TLS to connect on kafka")
	root.Flags().Bool(authBasicEnabledFlag, false, "Enable basic auth")
	root.Flags().StringSlice(authBasicCredentialsFlag, []string{}, "HTTP basic auth credentials (<username>:<password>)")
	root.Flags().Bool(authBearerEnabledFlag, false, "Enable bearer auth")
	root.Flags().String(authBearerIntrospectUrlFlag, "", "OAuth2 introspect URL")
	root.Flags().StringSlice(authBearerAudienceFlag, []string{}, "Allowed audiences")
	root.Flags().Bool(authBearerAudiencesWildcardFlag, false, "Don't check audience")
	root.Flags().Bool(authBearerUseScopesFlag, false, "Use scopes as defined by rfc https://datatracker.ietf.org/doc/html/rfc8693")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	err := viper.BindPFlags(root.Flags())
	if err != nil {
		panic(err)
	}

	return root
}

func Execute() {
	if err := rootCommand().Execute(); err != nil {
		if _, err := fmt.Fprintln(os.Stderr, err); err != nil {
			panic(err)
		}
		os.Exit(1)
	}
}
