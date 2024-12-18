package cmd

import (
	"errors"
	"log"
	"os"

	"github.com/bombsimon/logrusr/v3"
	sharedapi "github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/auth"
	"github.com/formancehq/go-libs/v2/bun/bunconnect"
	"github.com/formancehq/go-libs/v2/health"
	"github.com/formancehq/go-libs/v2/licence"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/otlp"
	"github.com/formancehq/go-libs/v2/otlp/otlpmetrics"
	"github.com/formancehq/go-libs/v2/otlp/otlptraces"
	"github.com/formancehq/go-libs/v2/profiling"
	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/go-libs/v2/temporal"
	"github.com/formancehq/payments/internal/api"
	v2 "github.com/formancehq/payments/internal/api/v2"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.uber.org/fx"
)

func setLogger() {
	// Add a dedicated logger for opentelemetry in case of error
	otel.SetLogger(logrusr.New(logrus.New().WithField("component", "otlp")))
}

func commonOptions(cmd *cobra.Command) (fx.Option, error) {
	configEncryptionKey, _ := cmd.Flags().GetString(ConfigEncryptionKeyFlag)
	if configEncryptionKey == "" {
		return nil, errors.New("missing config encryption key")
	}

	connectionOptions, err := bunconnect.ConnectionOptionsFromFlags(cmd)
	if err != nil {
		return nil, err
	}

	listen, _ := cmd.Flags().GetString(ListenFlag)
	stack, _ := cmd.Flags().GetString(StackFlag)
	stackPublicURL, _ := cmd.Flags().GetString(stackPublicURLFlag)
	debug, _ := cmd.Flags().GetBool(service.DebugFlag)
	jsonFormatter, _ := cmd.Flags().GetBool(logging.JsonFormattingLoggerFlag)
	temporalNamespace, _ := cmd.Flags().GetString(temporal.TemporalNamespaceFlag)
	temporalMaxConcurrentWorkflowTaskPollers, _ := cmd.Flags().GetInt(temporalMaxConcurrentWorkflowTaskPollersFlag)

	if len(os.Args) < 2 {
		// this shouldn't happen as long as this function is called by a subcommand
		log.Fatalf("os arguments does not contain command name: %s", os.Args)
	}
	rawFlags := os.Args[2:]

	return fx.Options(
		fx.Provide(func() *bunconnect.ConnectionOptions {
			return connectionOptions
		}),
		fx.Provide(func() sharedapi.ServiceInfo {
			return sharedapi.ServiceInfo{
				Version: Version,
			}
		}),
		otlp.FXModuleFromFlags(cmd),
		otlptraces.FXModuleFromFlags(cmd),
		otlpmetrics.FXModuleFromFlags(cmd),
		fx.Provide(metrics.RegisterMetricsRegistry),
		fx.Invoke(func(metrics.MetricsRegistry) {}),
		temporal.FXModuleFromFlags(
			cmd,
			engine.Tracer,
			temporal.SearchAttributes{
				SearchAttributes: engine.SearchAttributes,
			},
		),
		auth.FXModuleFromFlags(cmd),
		health.Module(),
		publish.FXModuleFromFlags(cmd, service.IsDebug(cmd)),
		licence.FXModuleFromFlags(cmd, ServiceName),
		storage.Module(cmd, *connectionOptions, configEncryptionKey),
		api.NewModule(listen, service.IsDebug(cmd)),
		profiling.FXModuleFromFlags(cmd),
		engine.Module(
			stack,
			stackPublicURL,
			temporalNamespace,
			temporalMaxConcurrentWorkflowTaskPollers,
			rawFlags,
			debug,
			jsonFormatter,
		),
		v2.NewModule(),
		v3.NewModule(),
	), nil
}
