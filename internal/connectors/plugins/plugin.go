package plugins

import (
	"fmt"

	"github.com/formancehq/go-libs/v2/otlp/otlpmetrics"
	"github.com/formancehq/go-libs/v2/otlp/otlptraces"
	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/payments/internal/models"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.uber.org/fx"
)

// TODO(polo): metrics
func NewPlugin(name string, pluginConstructorFn models.PluginConstructorFn) *cobra.Command {
	cmd := &cobra.Command{
		Use:          fmt.Sprintf("serve %s plugin", name),
		Aliases:      []string{name},
		Short:        fmt.Sprintf("Launch %s plugin server", name),
		SilenceUsage: true,
		RunE:         runServer(pluginConstructorFn),
	}

	service.AddFlags(cmd.Flags())
	otlpmetrics.AddFlags(cmd.Flags())
	otlptraces.AddFlags(cmd.Flags())
	return cmd
}

func runServer(pluginConstructorFn models.PluginConstructorFn) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// TODO initialise logger here?

		opts := fx.Options(
			fx.Provide(fx.Annotate(noop.NewMeterProvider, fx.As(new(metric.MeterProvider)))),
			fx.Decorate(fx.Annotate(func(meterProvider metric.MeterProvider) (metrics.MetricsRegistry, error) {
				return metrics.RegisterMetricsRegistry(meterProvider)
			})),
			fx.Provide(metrics.RegisterMetricsRegistry),
			fx.Provide(pluginConstructorFn, NewServer),
			fx.Invoke(func(Server) {}),
		)

		return service.New(cmd.OutOrStderr(), opts).Run(cmd)
	}
}
