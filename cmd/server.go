package cmd

import (
	"fmt"

	sharedapi "github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/auth"
	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/payments/internal/api"
	v2 "github.com/formancehq/payments/internal/api/v2"
	v3 "github.com/formancehq/payments/internal/api/v3"
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
	cmd.Flags().String(ListenFlag, ":8080", "Listen address")
	cmd.Flags().String(stackPublicURLFlag, "", "Stack public url")
	// MaxConcurrentWorkflowTaskPollers should not be set to a number < 2, otherwise
	// temporal will panic.
	// After meeting with the temporal team, we decided to set it to 20 as per
	// their recommendation.
	cmd.Flags().Int(temporalMaxConcurrentWorkflowTaskPollersFlag, 20, "Max concurrent workflow task pollers")
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
	), nil
}
