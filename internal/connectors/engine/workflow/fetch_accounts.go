package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type FetchNextAccounts struct {
	Config             models.Config      `json:"config"`
	ConnectorID        models.ConnectorID `json:"connectorID"`
	FromPayload        *FromPayload       `json:"fromPayload"`
	Periodically       bool               `json:"periodically"`
	AdditionalMetadata map[string]string  `json:"additionalMetadata"`
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

	hasMore := true
	for hasMore {
		accountsResponse, err := activities.PluginFetchNextAccounts(
			infiniteRetryContext(ctx),
			fetchNextAccount.ConnectorID,
			fetchNextAccount.FromPayload.GetPayload(),
			state.State,
			fetchNextAccount.Config.PageSize,
			fetchNextAccount.Periodically,
		)
		if err != nil {
			return errors.Wrap(err, "fetching next accounts")
		}

		accounts, err := models.FromPSPAccounts(
			accountsResponse.Accounts,
			models.ACCOUNT_TYPE_INTERNAL,
			fetchNextAccount.ConnectorID,
			fetchNextAccount.AdditionalMetadata,
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
		for _, account := range accounts {
			acc := account

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
					RunSendEvents,
					SendEvents{
						Account: &acc,
					},
				).Get(ctx, nil); err != nil {
					errChan <- errors.Wrap(err, "sending events")
				}
			})
		}

		for _, account := range accountsResponse.Accounts {
			acc := account

			wg.Add(1)
			workflow.Go(ctx, func(ctx workflow.Context) {
				defer wg.Done()

				payload, err := json.Marshal(acc)
				if err != nil {
					errChan <- errors.Wrap(err, "marshalling account")
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
					Run,
					fetchNextAccount.Config,
					fetchNextAccount.ConnectorID,
					&FromPayload{
						ID:      acc.Reference,
						Payload: payload,
					},
					nextTasks,
				).Get(ctx, nil); err != nil {
					errChan <- errors.Wrap(err, "running next workflow")
				}
			})
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
