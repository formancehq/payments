package plugins

import (
	"fmt"
	"os"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/otlp"
	"github.com/formancehq/go-libs/v2/otlp/otlpmetrics"
	"github.com/formancehq/go-libs/v2/otlp/otlptraces"
	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/fx"
)

func loggerOptions(cmd *cobra.Command) *hclog.LoggerOptions {
	jsonFormatter, _ := cmd.Flags().GetBool(logging.JsonFormattingLoggerFlag)

	level := hclog.Info
	if service.IsDebug(cmd) {
		level = hclog.Debug
	}
	return &hclog.LoggerOptions{
		Level:      level,
		Output:     os.Stderr,
		JSONFormat: jsonFormatter,
	}
}

// TODO(polo): metrics
func NewPlugin(name string, pluginConstructorFn models.PluginConstructorFn) *cobra.Command {
	cmd := &cobra.Command{
		Use:          fmt.Sprintf("serve %s plugin", name),
		Aliases:      []string{name},
		Short:        fmt.Sprintf("Launch %s plugin server", name),
		SilenceUsage: true,
	}

	service.AddFlags(cmd.Flags())
	otlpmetrics.AddFlags(cmd.Flags())
	otlptraces.AddFlags(cmd.Flags())

	hlogger := hclog.New(loggerOptions(cmd))
	hclog.SetDefault(hlogger)

	logger := logging.NewHcLogLoggerAdapter(hlogger, nil)
	cmd.RunE = runServer(pluginConstructorFn, logger)
	return cmd
}

func runServer(
	pluginConstructorFn models.PluginConstructorFn,
	logger logging.Logger,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		opts := fx.Options(
			otlp.FXModuleFromFlags(cmd),
			fx.Provide(fx.Annotate(noop.NewMeterProvider, fx.As(new(metric.MeterProvider)))),
			fx.Decorate(fx.Annotate(func(meterProvider metric.MeterProvider) (metrics.MetricsRegistry, error) {
				return metrics.RegisterMetricsRegistry(meterProvider)
			})),
			fx.Provide(metrics.RegisterMetricsRegistry),
			fx.Provide(pluginConstructorFn, NewServer),
			fx.Invoke(func(Server) {}),
		)

		return service.NewWithLogger(logger, opts).Run(cmd)
	}
}
