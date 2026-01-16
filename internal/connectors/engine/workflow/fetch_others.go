package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/workflow"
)

type FetchNextOthers struct {
	ConnectorID  models.ConnectorID `json:"connectorID"`
	Name         string             `json:"name"`
	FromPayload  *FromPayload       `json:"fromPayload"`
	Periodically bool               `json:"periodically"`
}

func (w Workflow) runFetchNextOthers(
	ctx workflow.Context,
	fetchNextOthers FetchNextOthers,
	nextTasks []models.ConnectorTaskTree,
) error {
	if err := w.createInstance(ctx, fetchNextOthers.ConnectorID); err != nil {
		return errors.Wrap(err, "creating instance")
	}
	err := w.fetchNextOthers(ctx, fetchNextOthers, nextTasks)
	return w.terminateInstance(ctx, fetchNextOthers.ConnectorID, err)
}

func (w Workflow) fetchNextOthers(
	ctx workflow.Context,
	fetchNextOthers FetchNextOthers,
	nextTasks []models.ConnectorTaskTree,
) error {
	stateReference := models.CAPABILITY_FETCH_OTHERS.String()
	if fetchNextOthers.FromPayload != nil {
		stateReference = fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_OTHERS.String(), fetchNextOthers.FromPayload.ID)
	}

	stateID := models.StateID{
		Reference:   stateReference,
		ConnectorID: fetchNextOthers.ConnectorID,
	}
	state, err := activities.StorageStatesGet(infiniteRetryContext(ctx), stateID)
	if err != nil {
		return fmt.Errorf("retrieving state %s: %v", stateID.String(), err)
	}

	// Get pageSize from registry using provider from ConnectorID (no DB call needed)
	pageSize, err := registry.GetPageSize(fetchNextOthers.ConnectorID.Provider)
	if err != nil {
		return fmt.Errorf("getting page size: %w", err)
	}

	hasMore := true
	for hasMore {
		othersResponse, err := activities.PluginFetchNextOthers(
			infiniteRetryWithLongTimeoutContext(ctx),
			fetchNextOthers.ConnectorID,
			fetchNextOthers.Name,
			fetchNextOthers.FromPayload.GetPayload(),
			state.State,
			int(pageSize),
			fetchNextOthers.Periodically,
		)
		if err != nil {
			return errors.Wrap(err, "fetching next others")
		}

		wg := workflow.NewWaitGroup(ctx)
		errChan := make(chan error, len(othersResponse.Others))

		if !IsRunNextTaskOptimizationsEnabled(ctx) {
			for _, other := range othersResponse.Others {
				o := other
				fromPayload := &FromPayload{
					ID:      o.ID,
					Payload: o.Other,
				}

				w.runNextTaskAsChildWorkflow(ctx, fetchNextOthers.ConnectorID, nextTasks, wg, fromPayload, errChan)
			}
		} else if len(nextTasks) > 0 {
			// First, we need to get the connector to check if it is scheduled for deletion
			// because if it is, we don't need to run the next tasks
			plugin, err := w.connectors.Get(fetchNextOthers.ConnectorID)
			if err != nil {
				return fmt.Errorf("getting connector: %w", err)
			}

			if !plugin.IsScheduledForDeletion() {
				for _, other := range othersResponse.Others {
					o := other

					wg.Add(1)
					workflow.Go(ctx, func(ctx workflow.Context) {
						defer wg.Done()

						if err := w.runNextTasksV3_1(
							ctx,
							fetchNextOthers.ConnectorID,
							&FromPayload{
								ID:      o.ID,
								Payload: o.Other,
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

		state.State = othersResponse.NewState
		err = activities.StorageStatesStore(
			infiniteRetryContext(ctx),
			*state,
		)
		if err != nil {
			return errors.Wrap(err, "storing state")
		}

		hasMore = othersResponse.HasMore

		if w.shouldContinueAsNew(ctx) {
			// If we have lots and lots of accounts, sometimes, we need to
			// continue as new to not exeed the maximum history size or length
			// of a workflow.
			return workflow.NewContinueAsNewError(
				ctx,
				RunFetchNextOthers,
				fetchNextOthers,
				nextTasks,
			)
		}
	}

	return nil
}

const RunFetchNextOthers = "FetchOthers"
