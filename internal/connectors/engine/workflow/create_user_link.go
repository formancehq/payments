package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/utils"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

// TODO(polo): create tests for this file

type CreateUserLink struct {
	TaskID      models.TaskID
	ConnectorID models.ConnectorID
	PsuID       uuid.UUID

	IdempotencyKey    *uuid.UUID
	ClientRedirectURL *string
}

func (w Workflow) runCreateUserLink(
	ctx workflow.Context,
	createUserLink CreateUserLink,
) error {
	link, err := w.createUserLink(
		infiniteRetryContext(ctx),
		createUserLink,
	)
	if err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			createUserLink.TaskID,
			&createUserLink.ConnectorID,
			err,
		)
		if errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return w.updateTaskSuccess(
		ctx,
		createUserLink.TaskID,
		&createUserLink.ConnectorID,
		link,
	)
}

func (w Workflow) createUserLink(
	ctx workflow.Context,
	createUserLink CreateUserLink,
) (string, error) {
	psu, err := activities.StoragePaymentServiceUsersGet(
		infiniteRetryContext(ctx),
		createUserLink.PsuID,
	)
	if err != nil {
		return "", err
	}

	bankBridge, err := activities.StoragePSUBankBridgesGet(
		infiniteRetryContext(ctx),
		createUserLink.PsuID,
		createUserLink.ConnectorID,
	)
	if err != nil {
		return "", err
	}

	id := uuid.New()
	if createUserLink.IdempotencyKey != nil {
		id = *createUserLink.IdempotencyKey
	}

	attempt := models.PSUBankBridgeConnectionAttempt{
		ID:          id,
		PsuID:       createUserLink.PsuID,
		ConnectorID: createUserLink.ConnectorID,
		CreatedAt:   workflow.Now(ctx),
		Status:      models.PSUBankBridgeConnectionAttemptStatusPending,
		State: models.CallbackState{
			Randomized: uuid.New().String(),
			AttemptID:  id,
		},
		ClientRedirectURL: createUserLink.ClientRedirectURL,
	}

	err = activities.StoragePSUBankBridgeConnectionAttemptsStore(
		infiniteRetryContext(ctx),
		attempt,
	)
	if err != nil {
		return "", err
	}

	webhookBaseURL, err := utils.GetWebhookBaseURL(w.stackPublicURL, createUserLink.ConnectorID)
	if err != nil {
		return "", fmt.Errorf("joining webhook base URL: %w", err)
	}

	formanceRedirectURL, err := w.getFormanceRedirectURL(createUserLink.ConnectorID)
	if err != nil {
		return "", fmt.Errorf("joining formance redirect URI: %w", err)
	}

	resp, err := activities.PluginCreateUserLink(
		infiniteRetryContext(ctx),
		createUserLink.ConnectorID,
		models.CreateUserLinkRequest{
			AttemptID:           attempt.ID.String(),
			PaymentServiceUser:  models.ToPSPPaymentServiceUser(psu),
			PSUBankBridge:       bankBridge,
			ClientRedirectURL:   createUserLink.ClientRedirectURL,
			FormanceRedirectURL: &formanceRedirectURL,
			CallBackState:       attempt.State.String(),
			WebhookBaseURL:      webhookBaseURL,
		},
	)
	if err != nil {
		return "", err
	}

	if resp.TemporaryLinkToken != nil {
		attempt.TemporaryToken = resp.TemporaryLinkToken
		err = activities.StoragePSUBankBridgeConnectionAttemptsStore(
			infiniteRetryContext(ctx),
			attempt,
		)
		if err != nil {
			return "", err
		}
	}

	return resp.Link, nil
}

var RunCreateUserLink = "RunCreateUserLink"
