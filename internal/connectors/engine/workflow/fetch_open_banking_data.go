package workflow

import (
	"fmt"
	"slices"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type FetchOpenBankingData struct {
	PsuID        uuid.UUID
	ConnectionID string
	ConnectorID  models.ConnectorID
	Config       models.Config
	DataToFetch  []models.OpenBankingDataToFetch
	FromPayload  *FromPayload
}

func (w Workflow) runFetchOpenBankingData(
	ctx workflow.Context,
	fetchOpenBankingData FetchOpenBankingData,
) error {
	if len(fetchOpenBankingData.DataToFetch) == 0 {
		return fmt.Errorf(
			"no data to fetch for psu %s, connection %s connector %s",
			fetchOpenBankingData.PsuID,
			fetchOpenBankingData.ConnectionID,
			fetchOpenBankingData.ConnectorID,
		)
	}

	wg := workflow.NewWaitGroup(ctx)

	if slices.Contains(fetchOpenBankingData.DataToFetch, models.OpenBankingDataToFetchAccountsAndBalances) {
		wg.Add(1)
		workflow.Go(ctx, w.startFetchNextAccountWorkflow(wg, fetchOpenBankingData))
	}

	if slices.Contains(fetchOpenBankingData.DataToFetch, models.OpenBankingDataToFetchPayments) {
		wg.Add(1)
		workflow.Go(ctx, w.startFetchNextPaymentsWorkflow(wg, fetchOpenBankingData))
	}

	wg.Wait(ctx)

	now := workflow.Now(ctx)

	err := activities.StorageOpenBankingConnectionsLastUpdatedAtUpdate(
		infiniteRetryContext(ctx),
		fetchOpenBankingData.PsuID,
		fetchOpenBankingData.ConnectorID,
		fetchOpenBankingData.ConnectionID,
		now,
	)
	if err != nil {
		return fmt.Errorf("updating open banking connection last updated at: %w", err)
	}

	sendEvent := SendEvents{
		UserConnectionDataSynced: &models.UserConnectionDataSynced{
			PsuID:        fetchOpenBankingData.PsuID,
			ConnectorID:  fetchOpenBankingData.ConnectorID,
			ConnectionID: fetchOpenBankingData.ConnectionID,
			At:           now,
		},
	}

	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				TaskQueue:         w.getDefaultTaskQueue(),
				ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
				SearchAttributes: map[string]interface{}{
					SearchAttributeStack: w.stack,
				},
			},
		),
		RunSendEvents,
		sendEvent,
	).Get(ctx, nil); err != nil {
		return fmt.Errorf("sending events: %w", err)
	}

	return nil
}

func (w Workflow) startFetchNextAccountWorkflow(wg workflow.WaitGroup, fetchOpenBankingData FetchOpenBankingData) func(ctx workflow.Context) {
	return func(ctx workflow.Context) {
		defer wg.Done()

		if err := workflow.ExecuteChildWorkflow(
			workflow.WithChildOptions(
				ctx,
				workflow.ChildWorkflowOptions{
					TaskQueue:         w.getDefaultTaskQueue(),
					ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
					SearchAttributes: map[string]interface{}{
						SearchAttributeStack: w.stack,
					},
				},
			),
			RunFetchNextAccounts,
			FetchNextAccounts{
				Config:       fetchOpenBankingData.Config,
				ConnectorID:  fetchOpenBankingData.ConnectorID,
				FromPayload:  fetchOpenBankingData.FromPayload,
				Periodically: false,
			},
			[]models.ConnectorTaskTree{
				{
					TaskType:     models.TASK_FETCH_BALANCES,
					Name:         "fetch_balances",
					Periodically: false,
					NextTasks:    []models.ConnectorTaskTree{},
				},
			},
		).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Error("failed to fetch accounts", "error", err)
		}
	}
}

func (w Workflow) startFetchNextPaymentsWorkflow(wg workflow.WaitGroup, fetchOpenBankingData FetchOpenBankingData) func(ctx workflow.Context) {
	return func(ctx workflow.Context) {
		defer wg.Done()

		if err := workflow.ExecuteChildWorkflow(
			workflow.WithChildOptions(
				ctx,
				workflow.ChildWorkflowOptions{
					TaskQueue:         w.getDefaultTaskQueue(),
					ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
					SearchAttributes: map[string]interface{}{
						SearchAttributeStack: w.stack,
					},
				},
			),
			RunFetchNextPayments,
			FetchNextPayments{
				Config:       fetchOpenBankingData.Config,
				ConnectorID:  fetchOpenBankingData.ConnectorID,
				FromPayload:  fetchOpenBankingData.FromPayload,
				Periodically: false,
			},
			[]models.ConnectorTaskTree{},
		).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Error("failed to fetch payments", "error", err)
		}
	}
}

func (w Workflow) startFetchNextBalancesWorkflow(wg workflow.WaitGroup, fetchOpenBankingData FetchOpenBankingData) func(ctx workflow.Context) {
	return func(ctx workflow.Context) {
		defer wg.Done()

		if err := workflow.ExecuteChildWorkflow(
			workflow.WithChildOptions(
				ctx,
				workflow.ChildWorkflowOptions{
					TaskQueue:         w.getDefaultTaskQueue(),
					ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
					SearchAttributes: map[string]interface{}{
						SearchAttributeStack: w.stack,
					},
				},
			),
			RunFetchNextBalances,
			FetchNextBalances{
				Config:       fetchOpenBankingData.Config,
				ConnectorID:  fetchOpenBankingData.ConnectorID,
				FromPayload:  fetchOpenBankingData.FromPayload,
				Periodically: false,
			},
			[]models.ConnectorTaskTree{},
		).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Error("failed to fetch balances", "error", err)
		}
	}
}

const RunFetchOpenBankingData = "RunFetchOpenBankingData"
