package cmd

import (
	"fmt"

	"github.com/formancehq/go-libs/v3/auth"
	"github.com/formancehq/go-libs/v3/service"
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

	cmd.Flags().String(stackPublicURLFlag, "", "Stack public url")

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
	stackPublicURL, _ := cmd.Flags().GetString(stackPublicURLFlag)
	return fx.Options(
		auth.FXModuleFromFlags(cmd),
		api.NewModule(listen, service.IsDebug(cmd)),
		v2.NewModule(),
		v3.NewModule(),
		engine.Module(stack, stackPublicURL, service.IsDebug(cmd)),
	), nil
}
