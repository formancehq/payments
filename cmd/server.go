package cmd

import (
	"fmt"

	sharedapi "github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/auth"
	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/payments/internal/api"
	v2 "github.com/formancehq/payments/internal/api/v2"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func newServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "serve",
		Aliases:      []string{"server"},
		Short:        "Launch api server",
		SilenceUsage: true,
		RunE:         runServer(),
	}
	commonFlags(cmd)
	return cmd
}

func runServer() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		setLogger()

		opts := []fx.Option{}
		commonOpts, err := commonOptions(cmd)
		if err != nil {
			return fmt.Errorf("failed to configure common options for server: %w", err)
		}
		opts = append(opts, commonOpts)

		serverOpts, err := serverOptions(cmd)
		if err != nil {
			return fmt.Errorf("failed to configure options for server: %w", err)
		}
		opts = append(opts, serverOpts)

		return service.New(cmd.OutOrStdout(), fx.Options(opts...)).Run(cmd)
	}
}

func serverOptions(cmd *cobra.Command) (fx.Option, error) {
	listen, _ := cmd.Flags().GetString(ListenFlag)
	stack, _ := cmd.Flags().GetString(StackFlag)
	return fx.Options(
		fx.Provide(func() sharedapi.ServiceInfo {
			return sharedapi.ServiceInfo{
				Version: Version,
			}
		}),
		auth.FXModuleFromFlags(cmd),
		api.NewModule(listen, service.IsDebug(cmd)),
		v2.NewModule(),
		v3.NewModule(),
		engine.Module(stack, service.IsDebug(cmd)),
	), nil
}
