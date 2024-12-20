package cmd

import (
	"errors"

	"github.com/bombsimon/logrusr/v3"
	"github.com/formancehq/go-libs/v2/auth"
	"github.com/formancehq/go-libs/v2/aws/iam"
	"github.com/formancehq/go-libs/v2/bun/bunconnect"
	"github.com/formancehq/go-libs/v2/health"
	"github.com/formancehq/go-libs/v2/licence"
	"github.com/formancehq/go-libs/v2/otlp"
	"github.com/formancehq/go-libs/v2/otlp/otlpmetrics"
	"github.com/formancehq/go-libs/v2/otlp/otlptraces"
	"github.com/formancehq/go-libs/v2/profiling"
	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/go-libs/v2/temporal"
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

func commonFlags(cmd *cobra.Command) {
	cmd.Flags().String(StackFlag, "", "Stack name")
	service.AddFlags(cmd.Flags())
	otlpmetrics.AddFlags(cmd.Flags())
	otlptraces.AddFlags(cmd.Flags())
	auth.AddFlags(cmd.Flags())
	publish.AddFlags(ServiceName, cmd.Flags())
	bunconnect.AddFlags(cmd.Flags())
	iam.AddFlags(cmd.Flags())
	profiling.AddFlags(cmd.Flags())
	temporal.AddFlags(cmd.Flags())
	licence.AddFlags(cmd.Flags())
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

	stack, _ := cmd.Flags().GetString(StackFlag)
	stackPublicURL, _ := cmd.Flags().GetString(stackPublicURLFlag)
	debug, _ := cmd.Flags().GetBool(service.DebugFlag)
	temporalNamespace, _ := cmd.Flags().GetString(temporal.TemporalNamespaceFlag)
	temporalMaxConcurrentWorkflowTaskPollers, _ := cmd.Flags().GetInt(temporalMaxConcurrentWorkflowTaskPollersFlag)

	return fx.Options(
		fx.Provide(func() *bunconnect.ConnectionOptions {
			return connectionOptions
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
		health.Module(),
		publish.FXModuleFromFlags(cmd, service.IsDebug(cmd)),
		licence.FXModuleFromFlags(cmd, ServiceName),
		storage.Module(cmd, *connectionOptions, configEncryptionKey),
		profiling.FXModuleFromFlags(cmd),
		engine.Module(
			stack,
			stackPublicURL,
			temporalNamespace,
			temporalMaxConcurrentWorkflowTaskPollers,
			debug,
		),
	), nil
}