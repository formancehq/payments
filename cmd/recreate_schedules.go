package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/spf13/cobra"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
)

func newRecreateSchedules() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "recreate-schedules",
		Short:        "Recreate missing Temporal schedules for active connectors",
		SilenceUsage: true,
		RunE:         runRecreateSchedules(),
	}
	commonFlags(cmd)
	return cmd
}

func runRecreateSchedules() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		setLogger()

		stack, _ := cmd.Flags().GetString(StackFlag)
		logger := logging.NewDefaultLogger(cmd.OutOrStdout(), true, true, false)

		var rs *RecreateSchedules
		commonOpts, err := commonOptions(cmd)
		if err != nil {
			return fmt.Errorf("failed to configure common options: %w", err)
		}

		options := []fx.Option{
			fx.Supply(fx.Annotate(logger, fx.As(new(logging.Logger)))),
			commonOpts,
			fx.Provide(func(logger logging.Logger, temporalClient client.Client, storage storage.Storage) *RecreateSchedules {
				return NewRecreateSchedules(logger, temporalClient, storage, stack)
			}),
			fx.Populate(&rs),
		}

		app := fx.New(options...)
		if err := app.Start(cmd.Context()); err != nil {
			return err
		}
		defer func() {
			if err := app.Stop(context.Background()); err != nil {
				logger.Errorf("failed to stop app: %s", err)
			}
		}()

		return rs.Run(cmd.Context())
	}
}

type RecreateSchedules struct {
	logger         logging.Logger
	temporalClient client.Client
	storage        storage.Storage
	stack          string
}

func NewRecreateSchedules(logger logging.Logger, temporalClient client.Client, storage storage.Storage, stack string) *RecreateSchedules {
	return &RecreateSchedules{
		logger:         logger,
		temporalClient: temporalClient,
		storage:        storage,
		stack:          stack,
	}
}

func (r *RecreateSchedules) Run(ctx context.Context) error {
	r.logger.Infof("recreating Temporal schedules for stack %q", r.stack)

	connectors, err := r.listActiveConnectors(ctx)
	if err != nil {
		return fmt.Errorf("failed to list connectors: %w", err)
	}

	r.logger.Infof("found %d active connector(s)", len(connectors))

	for _, connector := range connectors {
		if err := r.recreateConnectorSchedules(ctx, connector); err != nil {
			r.logger.Errorf("failed to recreate schedules for connector %s: %v", connector.ID.String(), err)
			// Continue with other connectors
			continue
		}
	}

	r.logger.Infof("done recreating schedules")
	return nil
}

func (r *RecreateSchedules) listActiveConnectors(ctx context.Context) ([]models.Connector, error) {
	var result []models.Connector
	query := storage.NewListConnectorsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.ConnectorQuery{}).
			WithPageSize(100),
	)

	for {
		page, err := r.storage.ConnectorsList(ctx, query)
		if err != nil {
			return nil, err
		}

		for _, c := range page.Data {
			if !c.ScheduledForDeletion {
				result = append(result, c)
			}
		}

		if !page.HasMore {
			break
		}

		if err := bunpaginate.UnmarshalCursor(page.Next, &query); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (r *RecreateSchedules) recreateConnectorSchedules(ctx context.Context, connector models.Connector) error {
	r.logger.Infof("processing connector %s (%s)", connector.Name, connector.ID.String())

	taskTree, err := r.storage.ConnectorTasksTreeGet(ctx, connector.ID)
	if err != nil {
		return fmt.Errorf("failed to get task tree: %w", err)
	}
	if taskTree == nil {
		r.logger.Infof("  no task tree found, skipping")
		return nil
	}

	var config models.Config
	if err := json.Unmarshal(connector.Config, &config); err != nil {
		return fmt.Errorf("failed to unmarshal connector config: %w", err)
	}

	if config.PollingPeriod == 0 {
		config.PollingPeriod = 30 * time.Minute
	}

	taskQueue := engine.GetDefaultTaskQueue(r.stack)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	r.walkTaskTree(ctx, *taskTree, connector.ID, config, taskQueue, nil, &wg, &mu, &errs)

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("encountered %d error(s) recreating schedules", len(errs))
	}

	return nil
}

func (r *RecreateSchedules) walkTaskTree(
	ctx context.Context,
	tasks []models.ConnectorTaskTree,
	connectorID models.ConnectorID,
	config models.Config,
	taskQueue string,
	fromPayload *workflow.FromPayload,
	wg *sync.WaitGroup,
	mu *sync.Mutex,
	errs *[]error,
) {
	for _, task := range tasks {
		if !task.Periodically {
			continue
		}

		workflowName, capability, request := r.buildScheduleParams(task, connectorID, fromPayload)
		if workflowName == "" {
			r.logger.Errorf("  unknown task type %d, skipping", task.TaskType)
			continue
		}

		var scheduleID string
		if fromPayload == nil {
			scheduleID = fmt.Sprintf("%s-%s-%s", r.stack, connectorID.String(), capability.String())
		} else {
			scheduleID = fmt.Sprintf("%s-%s-%s-%s", r.stack, connectorID.String(), capability.String(), fromPayload.ID)
		}

		nextTasks := task.NextTasks

		wg.Add(1)
		go func(scheduleID, workflowName string, request any, nextTasks []models.ConnectorTaskTree) {
			defer wg.Done()

			err := r.createSchedule(ctx, scheduleID, workflowName, config.PollingPeriod, taskQueue, request, nextTasks)
			if err != nil {
				r.logger.Errorf("  failed to create schedule %s: %v", scheduleID, err)
				mu.Lock()
				*errs = append(*errs, err)
				mu.Unlock()
				return
			}

			r.logger.Infof("  schedule %s ensured (workflow=%s, interval=%s)", scheduleID, workflowName, config.PollingPeriod)
		}(scheduleID, workflowName, request, nextTasks)
	}
}

func (r *RecreateSchedules) buildScheduleParams(
	task models.ConnectorTaskTree,
	connectorID models.ConnectorID,
	fromPayload *workflow.FromPayload,
) (workflowName string, capability models.Capability, request any) {
	switch task.TaskType {
	case models.TASK_FETCH_ACCOUNTS:
		return workflow.RunFetchNextAccounts, models.CAPABILITY_FETCH_ACCOUNTS, workflow.FetchNextAccounts{
			ConnectorID:  connectorID,
			FromPayload:  fromPayload,
			Periodically: true,
		}
	case models.TASK_FETCH_BALANCES:
		return workflow.RunFetchNextBalances, models.CAPABILITY_FETCH_BALANCES, workflow.FetchNextBalances{
			ConnectorID:  connectorID,
			FromPayload:  fromPayload,
			Periodically: true,
		}
	case models.TASK_FETCH_EXTERNAL_ACCOUNTS:
		return workflow.RunFetchNextExternalAccounts, models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS, workflow.FetchNextExternalAccounts{
			ConnectorID:  connectorID,
			FromPayload:  fromPayload,
			Periodically: true,
		}
	case models.TASK_FETCH_OTHERS:
		return workflow.RunFetchNextOthers, models.CAPABILITY_FETCH_OTHERS, workflow.FetchNextOthers{
			ConnectorID:  connectorID,
			Name:         task.Name,
			FromPayload:  fromPayload,
			Periodically: true,
		}
	case models.TASK_FETCH_PAYMENTS:
		return workflow.RunFetchNextPayments, models.CAPABILITY_FETCH_PAYMENTS, workflow.FetchNextPayments{
			ConnectorID:  connectorID,
			FromPayload:  fromPayload,
			Periodically: true,
		}
	case models.TASK_CREATE_WEBHOOKS:
		return workflow.RunCreateWebhooks, models.CAPABILITY_CREATE_WEBHOOKS, workflow.CreateWebhooks{
			ConnectorID: connectorID,
			FromPayload: fromPayload,
		}
	default:
		return "", 0, nil
	}
}

func (r *RecreateSchedules) createSchedule(
	ctx context.Context,
	scheduleID string,
	workflowName string,
	pollingPeriod time.Duration,
	taskQueue string,
	request any,
	nextTasks []models.ConnectorTaskTree,
) error {
	jitter := pollingPeriod / 2
	maxJitter := 5 * time.Minute
	if jitter > maxJitter {
		jitter = maxJitter
	}

	_, err := r.temporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{
				{Every: pollingPeriod},
			},
			Jitter: jitter,
		},
		Action: &client.ScheduleWorkflowAction{
			ID:       scheduleID,
			Workflow: workflowName,
			Args:     []any{request, nextTasks},
			TaskQueue: taskQueue,
			TypedSearchAttributes: temporal.NewSearchAttributes(
				temporal.NewSearchAttributeKeyKeyword(workflow.SearchAttributeScheduleID).ValueSet(scheduleID),
				temporal.NewSearchAttributeKeyKeyword(workflow.SearchAttributeStack).ValueSet(r.stack),
			),
		},
		Overlap:            enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
		TriggerImmediately: true,
		SearchAttributes: map[string]any{
			workflow.SearchAttributeScheduleID: scheduleID,
			workflow.SearchAttributeStack:      r.stack,
		},
	})

	if err != nil {
		var already *serviceerror.AlreadyExists
		var wfAlreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &wfAlreadyStarted) || errors.As(err, &already) {
			return nil
		}
		if errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
			return nil
		}
		return err
	}

	return nil
}
