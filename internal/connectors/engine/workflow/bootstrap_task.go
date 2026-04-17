package workflow

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// bootstrapPageDelay is the sleep between consecutive bootstrap pages. Kept
// low enough that a large paginated fetch still completes in reasonable time,
// but not zero so we don't hammer the upstream API.
const bootstrapPageDelay = 200 * time.Millisecond

type BootstrapTaskRequest struct {
	ConnectorID models.ConnectorID `json:"connectorID"`
	TaskType    models.TaskType    `json:"taskType"`
}

func (w Workflow) runBootstrapTask(
	ctx workflow.Context,
	req BootstrapTaskRequest,
) error {
	switch req.TaskType {
	case models.TASK_FETCH_ACCOUNTS:
		return w.bootstrapFetchAccounts(ctx, req.ConnectorID)
	default:
		return temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("bootstrap does not support task type %d", req.TaskType),
			ErrValidation,
			nil,
		)
	}
}

// bootstrapFetchAccounts paginates PluginFetchNextAccounts to completion and
// persists the new state after each page so a mid-bootstrap crash can resume
// from the last completed page. It does not fan out to downstream tasks —
// that happens once the periodic schedule starts, post-bootstrap.
func (w Workflow) bootstrapFetchAccounts(
	ctx workflow.Context,
	connectorID models.ConnectorID,
) error {
	stateID := models.StateID{
		Reference:   models.CAPABILITY_FETCH_ACCOUNTS.String(),
		ConnectorID: connectorID,
	}
	state, err := activities.StorageStatesGet(infiniteRetryContext(ctx), stateID)
	if err != nil {
		return fmt.Errorf("retrieving state %s: %w", stateID.String(), err)
	}

	pageSize, err := registry.GetPageSize(connectorID.Provider)
	if err != nil {
		return fmt.Errorf("getting page size: %w", err)
	}

	hasMore := true
	for hasMore {
		resp, err := activities.PluginFetchNextAccounts(
			infiniteRetryWithLongTimeoutContext(ctx),
			connectorID,
			nil,
			state.State,
			int(pageSize),
			false,
		)
		if err != nil {
			return errors.Wrap(err, "bootstrap fetching next accounts")
		}

		accounts, err := models.FromPSPAccounts(
			resp.Accounts,
			models.ACCOUNT_TYPE_INTERNAL,
			connectorID,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate accounts",
				ErrValidation,
				err,
			)
		}

		if len(accounts) > 0 {
			if err := activities.StorageAccountsStore(infiniteRetryContext(ctx), accounts); err != nil {
				return errors.Wrap(err, "bootstrap storing accounts")
			}
		}

		state.State = resp.NewState
		if err := activities.StorageStatesStore(infiniteRetryContext(ctx), *state); err != nil {
			return errors.Wrap(err, "bootstrap storing state")
		}

		hasMore = resp.HasMore
		if hasMore {
			if err := workflow.Sleep(ctx, bootstrapPageDelay); err != nil {
				return errors.Wrap(err, "bootstrap sleep between pages")
			}

			// Bootstrap may page through very large account sets. When the
			// Temporal history approaches its ceiling, continue as new so the
			// remaining pages run in a fresh history. Because the state is
			// persisted after every page, the new run resumes from the cursor
			// we just stored — no duplicate fetches.
			if w.shouldContinueAsNew(ctx) {
				return workflow.NewContinueAsNewError(
					ctx,
					RunBootstrapTask,
					BootstrapTaskRequest{
						ConnectorID: connectorID,
						TaskType:    models.TASK_FETCH_ACCOUNTS,
					},
				)
			}
		}
	}

	return nil
}

const RunBootstrapTask = "RunBootstrapTask"
