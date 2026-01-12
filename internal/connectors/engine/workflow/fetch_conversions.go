package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type FetchNextConversions struct {
	ConnectorID  models.ConnectorID `json:"connectorID"`
	FromPayload  *FromPayload       `json:"fromPayload"`
	Periodically bool               `json:"periodically"`
}

func (w Workflow) runFetchNextConversions(
	ctx workflow.Context,
	fetchNextConversions FetchNextConversions,
	nextTasks []models.ConnectorTaskTree,
) error {
	if err := w.createInstance(ctx, fetchNextConversions.ConnectorID); err != nil {
		return errors.Wrap(err, "creating instance")
	}
	err := w.fetchConversions(ctx, fetchNextConversions, nextTasks)
	return w.terminateInstance(ctx, fetchNextConversions.ConnectorID, err)
}

func (w Workflow) fetchConversions(
	ctx workflow.Context,
	fetchNextConversions FetchNextConversions,
	nextTasks []models.ConnectorTaskTree,
) error {
	stateReference := models.CAPABILITY_FETCH_CONVERSIONS.String()
	if fetchNextConversions.FromPayload != nil {
		stateReference = fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_CONVERSIONS.String(), fetchNextConversions.FromPayload.ID)
	}

	stateID := models.StateID{
		Reference:   stateReference,
		ConnectorID: fetchNextConversions.ConnectorID,
	}
	state, err := activities.StorageStatesGet(infiniteRetryContext(ctx), stateID)
	if err != nil {
		return fmt.Errorf("retrieving state %s: %v", stateID.String(), err)
	}

	// Get pageSize from registry using provider from ConnectorID (no DB call needed)
	pageSize, err := registry.GetPageSize(fetchNextConversions.ConnectorID.Provider)
	if err != nil {
		return fmt.Errorf("getting page size: %w", err)
	}

	hasMore := true
	for hasMore {
		conversionsResponse, err := activities.PluginFetchNextConversions(
			fetchNextActivityRetryContext(ctx),
			fetchNextConversions.ConnectorID,
			fetchNextConversions.FromPayload.GetPayload(),
			state.State,
			int(pageSize),
			fetchNextConversions.Periodically,
		)
		if err != nil {
			return errors.Wrap(err, "fetching next conversions")
		}

		conversions, err := models.FromPSPConversions(
			conversionsResponse.Conversions,
			fetchNextConversions.ConnectorID,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate psp conversions",
				ErrValidation,
				err,
			)
		}

		if len(conversionsResponse.Conversions) > 0 {
			err = activities.StorageConversionsUpsert(
				infiniteRetryContext(ctx),
				conversions,
			)
			if err != nil {
				return errors.Wrap(err, "storing next conversions")
			}
		}

		// TODO: Add event sending for conversions when needed
		// Currently conversions don't have event sending like accounts/balances

		state.State = conversionsResponse.NewState
		err = activities.StorageStatesStore(
			infiniteRetryContext(ctx),
			*state,
		)
		if err != nil {
			return errors.Wrap(err, "storing state")
		}

		hasMore = conversionsResponse.HasMore

		if w.shouldContinueAsNew(ctx) {
			return workflow.NewContinueAsNewError(
				ctx,
				RunFetchNextConversions,
				fetchNextConversions,
				nextTasks,
			)
		}
	}

	return nil
}

const RunFetchNextConversions = "FetchConversions"
