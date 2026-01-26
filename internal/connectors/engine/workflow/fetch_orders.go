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

type FetchNextOrders struct {
	ConnectorID  models.ConnectorID `json:"connectorID"`
	FromPayload  *FromPayload       `json:"fromPayload"`
	Periodically bool               `json:"periodically"`
}

func (w Workflow) runFetchNextOrders(
	ctx workflow.Context,
	fetchNextOrders FetchNextOrders,
	nextTasks []models.ConnectorTaskTree,
) error {
	if err := w.createInstance(ctx, fetchNextOrders.ConnectorID); err != nil {
		return errors.Wrap(err, "creating instance")
	}
	err := w.fetchOrders(ctx, fetchNextOrders, nextTasks)
	return w.terminateInstance(ctx, fetchNextOrders.ConnectorID, err)
}

func (w Workflow) fetchOrders(
	ctx workflow.Context,
	fetchNextOrders FetchNextOrders,
	nextTasks []models.ConnectorTaskTree,
) error {
	stateReference := models.CAPABILITY_FETCH_ORDERS.String()
	if fetchNextOrders.FromPayload != nil {
		stateReference = fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_ORDERS.String(), fetchNextOrders.FromPayload.ID)
	}

	stateID := models.StateID{
		Reference:   stateReference,
		ConnectorID: fetchNextOrders.ConnectorID,
	}
	state, err := activities.StorageStatesGet(infiniteRetryContext(ctx), stateID)
	if err != nil {
		return fmt.Errorf("retrieving state %s: %v", stateID.String(), err)
	}

	// Get pageSize from registry using provider from ConnectorID (no DB call needed)
	pageSize, err := registry.GetPageSize(fetchNextOrders.ConnectorID.Provider)
	if err != nil {
		return fmt.Errorf("getting page size: %w", err)
	}

	hasMore := true
	for hasMore {
		ordersResponse, err := activities.PluginFetchNextOrders(
			fetchNextActivityRetryContext(ctx),
			fetchNextOrders.ConnectorID,
			fetchNextOrders.FromPayload.GetPayload(),
			state.State,
			int(pageSize),
			fetchNextOrders.Periodically,
		)
		if err != nil {
			return errors.Wrap(err, "fetching next orders")
		}

		orders, err := models.FromPSPOrders(
			ordersResponse.Orders,
			fetchNextOrders.ConnectorID,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate psp orders",
				ErrValidation,
				err,
			)
		}

		if len(ordersResponse.Orders) > 0 {
			err = activities.StorageOrdersUpsert(
				infiniteRetryContext(ctx),
				orders,
			)
			if err != nil {
				return errors.Wrap(err, "storing next orders")
			}
		}

		// TODO: Add event sending for orders when needed
		// Currently orders don't have event sending like accounts/balances

		state.State = ordersResponse.NewState
		err = activities.StorageStatesStore(
			infiniteRetryContext(ctx),
			*state,
		)
		if err != nil {
			return errors.Wrap(err, "storing state")
		}

		hasMore = ordersResponse.HasMore

		if w.shouldContinueAsNew(ctx) {
			return workflow.NewContinueAsNewError(
				ctx,
				RunFetchNextOrders,
				fetchNextOrders,
				nextTasks,
			)
		}
	}

	return nil
}

const RunFetchNextOrders = "FetchOrders"
