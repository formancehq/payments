package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type UpdateUserLink struct {
	TaskID       models.TaskID
	ConnectorID  models.ConnectorID
	ConnectionID string
	PsuID        uuid.UUID

	IdempotencyKey    *uuid.UUID
	ClientRedirectURL *string
}

func (w Workflow) runUpdateUserLink(
	ctx workflow.Context,
	updateUserLink UpdateUserLink,
) error {
	link, err := w.updateUserLink(
		infiniteRetryContext(ctx),
		updateUserLink,
	)
	if err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			updateUserLink.TaskID,
			&updateUserLink.ConnectorID,
			err,
		)
		if errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return w.updateTaskSuccess(
		ctx,
		updateUserLink.TaskID,
		&updateUserLink.ConnectorID,
		link,
	)
}

func (w Workflow) updateUserLink(
	ctx workflow.Context,
	updateUserLink UpdateUserLink,
) (string, error) {
	psu, err := activities.StoragePaymentServiceUsersGet(
		infiniteRetryContext(ctx),
		updateUserLink.PsuID,
	)
	if err != nil {
		return "", err
	}

	bankBridge, err := activities.StoragePSUBankBridgesGet(
		infiniteRetryContext(ctx),
		updateUserLink.PsuID,
		updateUserLink.ConnectorID,
	)
	if err != nil {
		return "", err
	}

	connection, _, err := activities.StoragePSUBankBridgeConnectionsGetFromConnectionID(
		infiniteRetryContext(ctx),
		updateUserLink.ConnectorID,
		updateUserLink.ConnectionID,
	)
	if err != nil {
		return "", err
	}

	id := uuid.New()
	if updateUserLink.IdempotencyKey != nil {
		id = *updateUserLink.IdempotencyKey
	}

	attempt := models.PSUBankBridgeConnectionAttempt{
		ID:          id,
		PsuID:       updateUserLink.PsuID,
		ConnectorID: updateUserLink.ConnectorID,
		CreatedAt:   workflow.Now(ctx),
		Status:      models.PSUBankBridgeConnectionAttemptStatusPending,
		State: models.CallbackState{
			Randomized: uuid.New().String(),
			AttemptID:  id,
		},
		ClientRedirectURL: updateUserLink.ClientRedirectURL,
	}

	err = activities.StoragePSUBankBridgeConnectionAttemptsStore(
		infiniteRetryContext(ctx),
		attempt,
	)
	if err != nil {
		return "", err
	}

	formanceRedirectURL, err := w.getFormanceRedirectURL(updateUserLink.ConnectorID)
	if err != nil {
		return "", fmt.Errorf("joining formance redirect URI: %w", err)
	}

	return "", nil
}

var RunUpdateUserLink = "RunUpdateUserLink"
