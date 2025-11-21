package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type FetchNextExternalAccounts struct {
	Config       models.Config      `json:"config"`
	ConnectorID  models.ConnectorID `json:"connectorID"`
	FromPayload  *FromPayload       `json:"fromPayload"`
	Periodically bool               `json:"periodically"`
}

func (w Workflow) runFetchNextExternalAccounts(
	ctx workflow.Context,
	fetchNextExternalAccount FetchNextExternalAccounts,
	nextTasks []models.ConnectorTaskTree,
) error {
	if err := w.createInstance(ctx, fetchNextExternalAccount.ConnectorID); err != nil {
		return errors.Wrap(err, "creating instance")
	}
	err := w.fetchExternalAccounts(ctx, fetchNextExternalAccount, nextTasks)
	return w.terminateInstance(ctx, fetchNextExternalAccount.ConnectorID, err)
}

func (w Workflow) fetchExternalAccounts(
	ctx workflow.Context,
	fetchNextExternalAccount FetchNextExternalAccounts,
	nextTasks []models.ConnectorTaskTree,
) error {
	stateReference := models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS.String()
	if fetchNextExternalAccount.FromPayload != nil {
		stateReference = fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS.String(), fetchNextExternalAccount.FromPayload.ID)
	}

	stateID := models.StateID{
		Reference:   stateReference,
		ConnectorID: fetchNextExternalAccount.ConnectorID,
	}
	state, err := activities.StorageStatesGet(infiniteRetryContext(ctx), stateID)
	if err != nil {
		return fmt.Errorf("retrieving state %s: %v", stateID.String(), err)
	}

	hasMore := true
	for hasMore {
		externalAccountsResponse, err := activities.PluginFetchNextExternalAccounts(
			fetchNextActivityRetryContext(ctx),
			fetchNextExternalAccount.ConnectorID,
			fetchNextExternalAccount.FromPayload.GetPayload(),
			state.State,
			fetchNextExternalAccount.Config.PageSize,
			fetchNextExternalAccount.Periodically,
		)
		if err != nil {
			return errors.Wrap(err, "fetching next accounts")
		}

		accounts, err := models.FromPSPAccounts(
			externalAccountsResponse.ExternalAccounts,
			models.ACCOUNT_TYPE_EXTERNAL,
			fetchNextExternalAccount.ConnectorID,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate accounts",
				ErrValidation,
				err,
			)
		}

		if len(externalAccountsResponse.ExternalAccounts) > 0 {
			err = activities.StorageAccountsStore(
				infiniteRetryContext(ctx),
				accounts,
			)
			if err != nil {
				return errors.Wrap(err, "storing next accounts")
			}
		}

		wg := workflow.NewWaitGroup(ctx)
		errChan := make(chan error, len(externalAccountsResponse.ExternalAccounts))

		if len(nextTasks) > 0 {
			// First, we need to get the connector to check if it is scheduled for deletion
			// because if it is, we don't need to run the next tasks
			connector, err := activities.StorageConnectorsGet(infiniteRetryContext(ctx), fetchNextExternalAccount.ConnectorID)
			if err != nil {
				return fmt.Errorf("getting connector: %w", err)
			}

			if !connector.ScheduledForDeletion {
				for _, externalAccount := range externalAccountsResponse.ExternalAccounts {
					acc := externalAccount

					wg.Add(1)
					workflow.Go(ctx, func(ctx workflow.Context) {
						defer wg.Done()

						payload, err := json.Marshal(acc)
						if err != nil {
							errChan <- errors.Wrap(err, "marshalling external account")
							return
						}

						if err := w.runNextTasks(
							ctx,
							fetchNextExternalAccount.Config,
							connector,
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

		state.State = externalAccountsResponse.NewState
		err = activities.StorageStatesStore(
			infiniteRetryContext(ctx),
			*state,
		)
		if err != nil {
			return errors.Wrap(err, "storing state")
		}

		hasMore = externalAccountsResponse.HasMore

		if w.shouldContinueAsNew(ctx) {
			// If we have lots and lots of accounts, sometimes, we need to
			// continue as new to not exeed the maximum history size or length
			// of a workflow.
			return workflow.NewContinueAsNewError(
				ctx,
				RunFetchNextExternalAccounts,
				fetchNextExternalAccount,
				nextTasks,
			)
		}
	}

	return nil
}

const RunFetchNextExternalAccounts = "FetchExternalAccounts"
