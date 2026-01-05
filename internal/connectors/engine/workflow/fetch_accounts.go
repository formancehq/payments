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

type FetchNextAccounts struct {
	ConnectorID  models.ConnectorID `json:"connectorID"`
	FromPayload  *FromPayload       `json:"fromPayload"`
	Periodically bool               `json:"periodically"`
}

func (w Workflow) runFetchNextAccounts(
	ctx workflow.Context,
	fetchNextAccount FetchNextAccounts,
	nextTasks []models.ConnectorTaskTree,
) error {
	if err := w.createInstance(ctx, fetchNextAccount.ConnectorID); err != nil {
		return errors.Wrap(err, "creating instance")
	}
	err := w.fetchAccounts(ctx, fetchNextAccount, nextTasks)
	return w.terminateInstance(ctx, fetchNextAccount.ConnectorID, err)
}

func (w Workflow) fetchAccounts(
	ctx workflow.Context,
	fetchNextAccount FetchNextAccounts,
	nextTasks []models.ConnectorTaskTree,
) error {
	stateReference := models.CAPABILITY_FETCH_ACCOUNTS.String()
	if fetchNextAccount.FromPayload != nil {
		stateReference = fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_ACCOUNTS.String(), fetchNextAccount.FromPayload.ID)
	}

	stateID := models.StateID{
		Reference:   stateReference,
		ConnectorID: fetchNextAccount.ConnectorID,
	}
	state, err := activities.StorageStatesGet(infiniteRetryContext(ctx), stateID)
	if err != nil {
		return fmt.Errorf("retrieving state %s: %v", stateID.String(), err)
	}

	// Get pageSize from registry using provider from ConnectorID (no DB call needed)
	pageSize, err := registry.GetPageSize(fetchNextAccount.ConnectorID.Provider)
	if err != nil {
		return fmt.Errorf("getting page size: %w", err)
	}

	hasMore := true
	for hasMore {
		accountsResponse, err := activities.PluginFetchNextAccounts(
			fetchNextActivityRetryContext(ctx),
			fetchNextAccount.ConnectorID,
			fetchNextAccount.FromPayload.GetPayload(),
			state.State,
			int(pageSize),
			fetchNextAccount.Periodically,
		)
		if err != nil {
			return errors.Wrap(err, "fetching next accounts")
		}

		accounts, err := models.FromPSPAccounts(
			accountsResponse.Accounts,
			models.ACCOUNT_TYPE_INTERNAL,
			fetchNextAccount.ConnectorID,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate accounts",
				ErrValidation,
				err,
			)
		}

		if len(accountsResponse.Accounts) > 0 {
			err = activities.StorageAccountsStore(
				infiniteRetryContext(ctx),
				accounts,
			)
			if err != nil {
				return errors.Wrap(err, "storing next accounts")
			}
		}

		wg := workflow.NewWaitGroup(ctx)
		errChan := make(chan error, len(accountsResponse.Accounts)*2)

		if !IsEventOutboxPatternEnabled(ctx) {
			for _, account := range accounts {
				a := account

				sendEvents := SendEvents{
					Account: &a,
				}
				w.runSendEventAsChildWorkflow(ctx, wg, sendEvents, errChan)
			}
		}

		if !IsRunNextTaskAsActivityEnabled(ctx) {
			for _, account := range accountsResponse.Accounts {
				acc := account
				payload, err := json.Marshal(acc)
				if err != nil {
					errChan <- errors.Wrap(err, "marshalling account")
				}
				fromPayload := &FromPayload{
					ID:      acc.Reference,
					Payload: payload,
				}

				w.runNextTaskAsChildWorkflow(ctx, fetchNextAccount.ConnectorID, nextTasks, wg, fromPayload, errChan)
			}
		} else if len(nextTasks) > 0 {
			// First, we need to get the connector to check if it is scheduled for deletion
			// because if it is, we don't need to run the next tasks
			plugin, err := w.connectors.Get(fetchNextAccount.ConnectorID)
			if err != nil {
				return fmt.Errorf("getting connector: %w", err)
			}

			if !plugin.IsScheduledForDeletion() {
				for _, account := range accountsResponse.Accounts {
					acc := account

					wg.Add(1)
					workflow.Go(ctx, func(ctx workflow.Context) {
						defer wg.Done()

						payload, err := json.Marshal(acc)
						if err != nil {
							errChan <- errors.Wrap(err, "marshalling account")
							// don't continue if we can't marshal the account
							return
						}

						if err := w.runNextTasksV3_1(
							ctx,
							fetchNextAccount.ConnectorID,
							&FromPayload{
								ID:      acc.Reference,
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

		state.State = accountsResponse.NewState
		err = activities.StorageStatesStore(
			infiniteRetryContext(ctx),
			*state,
		)
		if err != nil {
			return errors.Wrap(err, "storing state")
		}

		hasMore = accountsResponse.HasMore

		if w.shouldContinueAsNew(ctx) {
			// If we have lots and lots of accounts, sometimes, we need to
			// continue as new to not exeed the maximum history size or length
			// of a workflow.
			return workflow.NewContinueAsNewError(
				ctx,
				RunFetchNextAccounts,
				fetchNextAccount,
				nextTasks,
			)
		}
	}

	return nil
}

const RunFetchNextAccounts = "FetchAccounts"
