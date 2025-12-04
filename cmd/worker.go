package cmd

import (
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/service"
	"github.com/formancehq/go-libs/v3/temporal"
	"github.com/formancehq/payments/internal/worker"
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
	// MaxConcurrentWorkflowTaskPollers should not be set to a number < 2, otherwise
	// temporal will panic.
	// After meeting with the temporal team, we decided to set it to 20 as per
	// their recommendation.
	cmd.Flags().Int(temporalMaxConcurrentWorkflowTaskPollersFlag, 4, "Max concurrent workflow task pollers")
	cmd.Flags().Int(temporalMaxConcurrentActivityTaskPollersFlag, 4, "Max concurrent activity task pollers")
	cmd.Flags().Int(temporalMaxSlotsPerPollerFlag, 10, "Max slot count per poller")
	cmd.Flags().Int(temporalMaxLocalActivitySlotsFlag, 50, "Max local activity slots")
	cmd.Flags().String(stackPublicURLFlag, "", "Stack public url")
	cmd.Flags().Duration(temporalRateLimitingRetryDelay, 5*time.Second, "Additional delay before a rate limited request is retried by Temporal workers")
	cmd.Flags().Bool(SkipOutboxScheduleCreationFlag, false, "Skip creating the outbox event publisher schedule (e.g. for tests)")
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
	listen, _ := cmd.Flags().GetString(ListenFlag)
	stack, _ := cmd.Flags().GetString(StackFlag)
	stackPublicURL, _ := cmd.Flags().GetString(stackPublicURLFlag)
	temporalNamespace, _ := cmd.Flags().GetString(temporal.TemporalNamespaceFlag)
	temporalRateLimitingRetryDelay, _ := cmd.Flags().GetDuration(temporalRateLimitingRetryDelay)
	temporalMaxConcurrentWorkflowTaskPollers, _ := cmd.Flags().GetInt(temporalMaxConcurrentWorkflowTaskPollersFlag)
	temporalMaxConcurrentActivityTaskPollers, _ := cmd.Flags().GetInt(temporalMaxConcurrentActivityTaskPollersFlag)
	temporalMaxSlotsPerPoller, _ := cmd.Flags().GetInt(temporalMaxSlotsPerPollerFlag)
	temporalMaxLocalActivitySlots, _ := cmd.Flags().GetInt(temporalMaxLocalActivitySlotsFlag)

	skipOutboxScheduleCreation, _ := cmd.Flags().GetBool(SkipOutboxScheduleCreationFlag)

	pollingPeriodDefault, _ := cmd.Flags().GetDuration(ConnectorPollingPeriodDefault)
	pollingPeriodMinimum, _ := cmd.Flags().GetDuration(ConnectorPollingPeriodMinimum)
	return fx.Options(
		worker.NewHealthCheckModule(listen, service.IsDebug(cmd)),
		worker.NewModule(
			stack,
			stackPublicURL,
			temporalNamespace,
			temporalRateLimitingRetryDelay,
			temporalMaxConcurrentWorkflowTaskPollers,
			temporalMaxConcurrentActivityTaskPollers,
			temporalMaxSlotsPerPoller,
			temporalMaxLocalActivitySlots,
			service.IsDebug(cmd),
			skipOutboxScheduleCreation,
			pollingPeriodDefault,
			pollingPeriodMinimum,
		),
	), nil
}
