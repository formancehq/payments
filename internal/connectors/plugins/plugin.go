package plugins

import (
	"fmt"

	"github.com/formancehq/go-libs/otlp/otlpmetrics"
	"github.com/formancehq/go-libs/otlp/otlptraces"
	"github.com/formancehq/go-libs/service"
	"github.com/formancehq/payments/internal/models"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

// TODO(polo): metrics
func NewPlugin(name string, pluginFn models.PluginFn) *cobra.Command {
	cmd := &cobra.Command{
		Use:          fmt.Sprintf("serve %s plugin", name),
		Aliases:      []string{name},
		Short:        fmt.Sprintf("Launch %s plugin server", name),
		SilenceUsage: true,
		RunE:         runServer(pluginFn),
	}

	service.AddFlags(cmd.Flags())
	otlpmetrics.AddFlags(cmd.Flags())
	otlptraces.AddFlags(cmd.Flags())
	return cmd
}

func runServer(pluginFn models.PluginFn) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// TODO initialise logger here?
		opts := fx.Options(
			fx.Provide(pluginFn, NewServer),
			fx.Invoke(func(Server) {}),
		)
		return service.New(cmd.OutOrStderr(), opts).Run(cmd)
	}
}
