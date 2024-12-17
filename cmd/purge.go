package cmd

import (
	"context"
	"fmt"
	"sync"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/temporal"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

func newPurge() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "purge",
		Short:        "Launch purge command to clean temporal",
		SilenceUsage: true,
		RunE:         runPurge(),
	}

	temporal.AddFlags(cmd.Flags())

	return cmd
}

func runPurge() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		setLogger()

		return purgeOptions(cmd)
	}
}

func purgeOptions(cmd *cobra.Command) error {
	temporalNamespace, _ := cmd.Flags().GetString(temporal.TemporalNamespaceFlag)
	stack, _ := cmd.Flags().GetString(StackFlag)
	logger := logging.NewDefaultLogger(cmd.OutOrStdout(), true, true, false)

	var purge *Purge
	options := []fx.Option{
		fx.Supply(fx.Annotate(logger, fx.As(new(logging.Logger)))),
		temporal.FXModuleFromFlags(
			cmd,
			engine.Tracer,
			temporal.SearchAttributes{
				SearchAttributes: engine.SearchAttributes,
			},
		),
		fx.Provide(func() metric.MeterProvider {
			return noop.NewMeterProvider()
		}),
		fx.Provide(NewPurge),
		fx.Populate(&purge),
	}

	app := fx.New(options...)
	if err := app.Start(cmd.Context()); err != nil {
		return err
	}
	defer app.Stop(context.Background())

	return purge.clean(cmd.Context(), temporalNamespace, stack)
}

type Purge struct {
	logger         logging.Logger
	temporalClient client.Client
}

func NewPurge(logger logging.Logger, temporalClient client.Client) *Purge {
	return &Purge{
		logger:         logger,
		temporalClient: temporalClient,
	}
}

func (p *Purge) clean(
	ctx context.Context,
	temporalNamespace string,
	stackName string,
) error {
	if err := p.cleanTemporalSchedules(ctx, stackName); err != nil {
		return err
	}

	if err := p.cleanTemporalWorkflows(ctx, temporalNamespace, stackName); err != nil {
		return err
	}

	return nil
}

func (p *Purge) cleanTemporalSchedules(
	ctx context.Context,
	stackName string,
) error {
	// list schedules
	listView, err := p.temporalClient.ScheduleClient().List(ctx, client.ScheduleListOptions{
		PageSize: 500,
		Query:    fmt.Sprintf("Stack=\"%s\"", stackName),
	})
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	for listView.HasNext() {
		s, err := listView.Next()
		if err != nil {
			return err
		}

		wg.Add(1)
		go func(s *client.ScheduleListEntry) {
			defer wg.Done()

			// get handle
			handle := p.temporalClient.ScheduleClient().GetHandle(ctx, s.ID)

			// delete schedule
			if err := handle.Delete(ctx); err != nil {
				p.logger.Errorf("failed to delete schedule %s: %v", s.ID, err)
				return
			}
		}(s)
	}

	wg.Wait()

	return nil
}

func (p *Purge) cleanTemporalWorkflows(
	ctx context.Context,
	temporalNamespace string,
	stackName string,
) error {
	var nextPageToken []byte
	wg := sync.WaitGroup{}
	for {
		resp, err := p.temporalClient.WorkflowService().ListWorkflowExecutions(
			ctx,
			&workflowservice.ListWorkflowExecutionsRequest{
				Namespace:     temporalNamespace,
				PageSize:      1000,
				NextPageToken: nextPageToken,
				Query:         fmt.Sprintf("Stack=\"%s\"", stackName),
			},
		)
		if err != nil {
			return err
		}

		toClean := 0
		for _, e := range resp.Executions {
			if e.Status != enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
				continue
			}

			toClean++
			wg.Add(1)
			go func(e *workflow.WorkflowExecutionInfo) {
				defer wg.Done()

				// close workflow
				_, err := p.temporalClient.WorkflowService().TerminateWorkflowExecution(
					ctx,
					&workflowservice.TerminateWorkflowExecutionRequest{
						Namespace:         temporalNamespace,
						WorkflowExecution: e.Execution,
						Reason:            "stack delete",
					},
				)
				if err != nil {
					p.logger.Errorf("failed to terminate workflow %s: %v", e.Execution.GetWorkflowId(), err)
					return
				}
			}(e)
		}

		if resp.NextPageToken == nil {
			break
		}

		nextPageToken = resp.NextPageToken
		p.logger.Infof("cleaned %d/1000 workflows, fetching next page", toClean)
	}

	wg.Wait()

	return nil
}
