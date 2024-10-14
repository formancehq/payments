package plugins

import (
	"context"
	"fmt"
	"sync"

	"github.com/formancehq/go-libs/otlp/otlpmetrics"
	"github.com/formancehq/go-libs/otlp/otlptraces"
	"github.com/formancehq/go-libs/service"
	"github.com/formancehq/payments/internal/connectors/grpc"
	"github.com/formancehq/payments/internal/models"
	"github.com/hashicorp/go-plugin"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

type Server interface{}

type server struct {
	plugin models.Plugin
}

func NewServer(lc fx.Lifecycle, shutdowner fx.Shutdowner, plg models.Plugin) Server {
	srv := &server{plugin: plg}
	wg := &sync.WaitGroup{}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			wg.Add(1)
			go func() {
				defer wg.Done()
				plugin.Serve(&plugin.ServeConfig{
					HandshakeConfig: grpc.Handshake,
					Plugins: map[string]plugin.Plugin{
						"psp": &grpc.PSPGRPCPlugin{Impl: NewGRPCImplem(srv.plugin)},
					},

					// A non-nil value here enables gRPC serving for this plugin...
					GRPCServer: plugin.DefaultGRPCServer,
				})
				// when Serve has ended the server closed usually because the plugin.Client told it to
				// if the parent application (managed by fx) is still running we need to tell it the plugin is done
				shutdowner.Shutdown(fx.ExitCode(0))
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// plugin.Serve is expected to block until the plugin.Client tells it to stop
			// this ensures the main plugin process doesn't exit before the plugin server shutsdown
			wg.Wait()
			return nil
		},
	})
	return srv
}

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
