package cmd

import (
	"fmt"

	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/go-libs/v2/temporal"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func newWorker() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "run-worker",
		Aliases:      []string{"worker"},
		Short:        "Launch api worker",
		SilenceUsage: true,
		RunE:         runWorker(),
	}
	commonFlags(cmd)
	cmd.Flags().String(stackPublicURLFlag, "", "Stack public url")
	// MaxConcurrentWorkflowTaskPollers should not be set to a number < 2, otherwise
	// temporal will panic.
	// After meeting with the temporal team, we decided to set it to 20 as per
	// their recommendation.
	cmd.Flags().Int(temporalMaxConcurrentWorkflowTaskPollersFlag, 20, "Max concurrent workflow task pollers")
	return cmd
}

func runWorker() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		setLogger()

		opts := []fx.Option{}
		commonOpts, err := commonOptions(cmd)
		if err != nil {
			return fmt.Errorf("failed to configure common options for worker: %w", err)
		}
		opts = append(opts, commonOpts)

		workerOpts, err := workerOptions(cmd)
		if err != nil {
			return fmt.Errorf("failed to configure options for worker: %w", err)
		}
		opts = append(opts, workerOpts)

		return service.New(cmd.OutOrStdout(), fx.Options(opts...)).Run(cmd)
	}
}

func workerOptions(cmd *cobra.Command) (fx.Option, error) {
	stack, _ := cmd.Flags().GetString(StackFlag)
	stackPublicURL, _ := cmd.Flags().GetString(stackPublicURLFlag)
	temporalNamespace, _ := cmd.Flags().GetString(temporal.TemporalNamespaceFlag)
	temporalMaxConcurrentWorkflowTaskPollers, _ := cmd.Flags().GetInt(temporalMaxConcurrentWorkflowTaskPollersFlag)
	return fx.Options(
		engine.WorkerModule(
			stack,
			stackPublicURL,
			temporalNamespace,
			temporalMaxConcurrentWorkflowTaskPollers,
			service.IsDebug(cmd),
		),
	), nil
}