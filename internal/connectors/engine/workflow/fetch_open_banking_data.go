package workflow

import (
	"fmt"

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
	FromPayload  *FromPayload
}

func (w Workflow) runFetchOpenBankingData(
	ctx workflow.Context,
	fetchOpenBankingData FetchOpenBankingData,
) error {
	additionalMetadata := map[string]string{
		models.ObjectConnectionIDMetadataKey: fetchOpenBankingData.ConnectionID,
		models.ObjectPSUIDMetadataKey:        fetchOpenBankingData.PsuID.String(),
	}

	wg := workflow.NewWaitGroup(ctx)

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
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
				Config:             fetchOpenBankingData.Config,
				ConnectorID:        fetchOpenBankingData.ConnectorID,
				FromPayload:        fetchOpenBankingData.FromPayload,
				Periodically:       false,
				AdditionalMetadata: additionalMetadata,
			},
			[]models.ConnectorTaskTree{},
		).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Error("failed to fetch accounts", "error", err)
		}
	})

	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
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
				Config:             fetchOpenBankingData.Config,
				ConnectorID:        fetchOpenBankingData.ConnectorID,
				FromPayload:        fetchOpenBankingData.FromPayload,
				Periodically:       false,
				AdditionalMetadata: additionalMetadata,
			},
			[]models.ConnectorTaskTree{},
		).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Error("failed to fetch payments", "error", err)
		}
	})

	wg.Wait(ctx)

	now := workflow.Now(ctx)

	err := activities.StoragePSUOpenBankingConnectionsLastUpdatedAtUpdate(
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

const RunFetchOpenBankingData = "RunFetchOpenBankingData"
