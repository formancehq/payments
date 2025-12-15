package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type FetchNextBalances struct {
	Config       models.Config      `json:"config"`
	ConnectorID  models.ConnectorID `json:"connectorID"`
	FromPayload  *FromPayload       `json:"fromPayload"`
	Periodically bool               `json:"periodically"`
}

func (w Workflow) runFetchNextBalances(
	ctx workflow.Context,
	fetchNextBalances FetchNextBalances,
	nextTasks []models.ConnectorTaskTree,
) error {
	if err := w.createInstance(ctx, fetchNextBalances.ConnectorID); err != nil {
		return errors.Wrap(err, "creating instance")
	}
	err := w.fetchBalances(ctx, fetchNextBalances, nextTasks)
	return w.terminateInstance(ctx, fetchNextBalances.ConnectorID, err)
}

func (w Workflow) fetchBalances(
	ctx workflow.Context,
	fetchNextBalances FetchNextBalances,
	nextTasks []models.ConnectorTaskTree,
) error {
	stateReference := models.CAPABILITY_FETCH_BALANCES.String()
	if fetchNextBalances.FromPayload != nil {
		stateReference = fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_BALANCES.String(), fetchNextBalances.FromPayload.ID)
	}

	stateID := models.StateID{
		Reference:   stateReference,
		ConnectorID: fetchNextBalances.ConnectorID,
	}
	state, err := activities.StorageStatesGet(infiniteRetryContext(ctx), stateID)
	if err != nil {
		return fmt.Errorf("retrieving state %s: %v", stateID.String(), err)
	}

	// Get pageSize from registry using provider from ConnectorID (no DB call needed)
	pageSize, err := registry.GetPageSize(fetchNextBalances.ConnectorID.Provider)
	if err != nil {
		return fmt.Errorf("getting page size: %w", err)
	}

	hasMore := true
	for hasMore {
		balancesResponse, err := activities.PluginFetchNextBalances(
			fetchNextActivityRetryContext(ctx),
			fetchNextBalances.ConnectorID,
			fetchNextBalances.FromPayload.GetPayload(),
			state.State,
			int(pageSize),
			fetchNextBalances.Periodically,
		)
		if err != nil {
			return errors.Wrap(err, "fetching next balances")
		}

		balances, err := models.FromPSPBalances(
			balancesResponse.Balances,
			fetchNextBalances.ConnectorID,
			nil,
			nil,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate psp balances",
				ErrValidation,
				err,
			)
		}

		if len(balancesResponse.Balances) > 0 {
			err = activities.StorageBalancesStore(
				infiniteRetryContext(ctx),
				balances,
			)
			if err != nil {
				return errors.Wrap(err, "storing next balances")
			}
		}

		wg := workflow.NewWaitGroup(ctx)
		errChan := make(chan error, len(balancesResponse.Balances))

		if len(nextTasks) > 0 {
			// First, we need to get the connector to check if it is scheduled for deletion
			// because if it is, we don't need to run the next tasks
			connectorMetadata, err := activities.StorageConnectorsGetMetadata(infiniteRetryContext(ctx), fetchNextBalances.ConnectorID)
			if err != nil {
				return fmt.Errorf("getting connector: %w", err)
			}

			if !connectorMetadata.ScheduledForDeletion {
				for _, balance := range balancesResponse.Balances {
					b := balance

					wg.Add(1)
					workflow.Go(ctx, func(ctx workflow.Context) {
						defer wg.Done()

						payload, err := json.Marshal(b)
						if err != nil {
							errChan <- errors.Wrap(err, "marshalling account")
							return
						}

						if err := w.runNextTasks(
							ctx,
							fetchNextBalances.Config,
							connectorMetadata,
							&FromPayload{
								ID:      fmt.Sprintf("%s-balances", b.AccountReference),
								Payload: payload,
							},
							nextTasks,
						); err != nil {
							errChan <- errors.Wrap(err, "running next tasks")
						}
					})
				}
			}
		}

		wg.Wait(ctx)
		close(errChan)
		for err := range errChan {
			if err != nil {
				return err
			}
		}

		state.State = balancesResponse.NewState
		err = activities.StorageStatesStore(
			infiniteRetryContext(ctx),
			*state,
		)
		if err != nil {
			return errors.Wrap(err, "storing state")
		}

		hasMore = balancesResponse.HasMore

		if w.shouldContinueAsNew(ctx) {
			// If we have lots and lots of accounts, sometimes, we need to
			// continue as new to not exeed the maximum history size or length
			// of a workflow.
			return workflow.NewContinueAsNewError(
				ctx,
				RunFetchNextBalances,
				fetchNextBalances,
				nextTasks,
			)
		}
	}

	return nil
}

const RunFetchNextBalances = "FetchBalances"
