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
	"go.uber.org/fx"
)

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

func runServer(
	pluginConstructorFn models.PluginConstructorFn,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		hlogger := hclog.New(loggerOptions())
		hclog.SetDefault(hlogger)

		opts := make([]fx.Option, 0)
		opts = append(opts,
			otlp.FXModuleFromFlags(cmd),
			otlpmetrics.FXModuleFromFlags(cmd),
			fx.Provide(metrics.RegisterMetricsRegistry),
			fx.Invoke(func(metrics.MetricsRegistry) {}),
		)

		opts = append(opts,
			fx.Provide(pluginConstructorFn, func() hclog.Logger { return hlogger }, NewServer),
			fx.Invoke(func(Server) {}),
		)

		logger := logging.NewHcLogLoggerAdapter(hlogger, nil)
		return service.NewWithLogger(logger, fx.Options(opts...)).Run(cmd)
	}
}

func loggerOptions() *hclog.LoggerOptions {
	// client-side logger settings (internal/connectors/engine/plugins/plugin.go) will override
	// the output format and verbosity, so we can just make this as verbose as possible
	return &hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	}
}
