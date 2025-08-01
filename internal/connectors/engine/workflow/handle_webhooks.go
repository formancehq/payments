package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type HandleWebhooks struct {
	ConnectorID models.ConnectorID
	URL         string
	URLPath     string
	Webhook     models.Webhook
	Config      *models.WebhookConfig
}

func (w Workflow) runHandleWebhooks(
	ctx workflow.Context,
	handleWebhooks HandleWebhooks,
) error {
	err := activities.StorageWebhooksStore(infiniteRetryContext(ctx), handleWebhooks.Webhook)
	if err != nil {
		return fmt.Errorf("storing webhook: %w", err)
	}

	resp, err := activities.PluginTranslateWebhook(
		infiniteRetryContext(ctx),
		handleWebhooks.ConnectorID,
		models.TranslateWebhookRequest{
			Name: handleWebhooks.Config.Name,
			Webhook: models.PSPWebhook{
				BasicAuth:   handleWebhooks.Webhook.BasicAuth,
				QueryValues: handleWebhooks.Webhook.QueryValues,
				Headers:     handleWebhooks.Webhook.Headers,
				Body:        handleWebhooks.Webhook.Body,
			},
			Config: handleWebhooks.Config,
		},
	)
	if err != nil {
		return fmt.Errorf("translating webhook: %w", err)
	}

	for _, response := range resp.Responses {
		switch {
		case response.DataReadyToFetch != nil:
			if err := w.handleTransactionReadyToFetchWebhook(ctx, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling bank bridge webhook: %w", err)
			}

		case response.UserLinkSessionFinished != nil:
			if err := w.handleUserLinkSessionFinishedWebhook(ctx, response); err != nil {
				return fmt.Errorf("handling user link session finished webhook: %w", err)
			}

		case response.UserConnectionDisconnected != nil:
			if err := w.handleUserDisconnectedWebhook(ctx, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling user disconnected webhook: %w", err)
			}

		case response.UserConnectionPendingDisconnect != nil:
			if err := w.handleUserPendingDisconnectWebhook(ctx, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling user pending disconnect webhook: %w", err)
			}

		default:
			// Default case, all the other webhooks are to store data
			if err := w.handleDataToStoreWebhook(ctx, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling data to store webhook: %w", err)
			}

		}
	}

	return nil
}

func (w Workflow) handleDataToStoreWebhook(
	ctx workflow.Context,
	handleWebhooks HandleWebhooks,
	response models.WebhookResponse,
) error {
	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				WorkflowID:            fmt.Sprintf("store-webhook-%s-%s-%s", w.stack, handleWebhooks.ConnectorID.String(), handleWebhooks.Webhook.ID),
				TaskQueue:             w.getDefaultTaskQueue(),
				ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
				WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
				SearchAttributes: map[string]interface{}{
					SearchAttributeStack: w.stack,
				},
			},
		),
		RunStoreWebhookTranslation,
		StoreWebhookTranslation{
			ConnectorID:     handleWebhooks.ConnectorID,
			Account:         response.Account,
			ExternalAccount: response.ExternalAccount,
			Payment:         response.Payment,
		},
	).Get(ctx, nil); err != nil {
		applicationError := &temporal.ApplicationError{}
		if errors.As(err, &applicationError) {
			if applicationError.Type() != "ChildWorkflowExecutionAlreadyStartedError" {
				return err
			}
		} else {
			return fmt.Errorf("storing webhook translation: %w", err)
		}
	}

	return nil
}

func (w Workflow) handleTransactionReadyToFetchWebhook(
	ctx workflow.Context,
	handleWebhooks HandleWebhooks,
	response models.WebhookResponse,
) error {
	connector, err := activities.StorageConnectorsGet(
		infiniteRetryContext(ctx),
		handleWebhooks.ConnectorID,
	)
	if err != nil {
		return fmt.Errorf("getting connector: %w", err)
	}

	var conn *models.PSUBankBridgeConnection
	var ba *models.PSUBankBridge
	var psuID uuid.UUID
	var connectionID string
	if response.DataReadyToFetch.ID != nil {
		connection, psu, err := activities.StoragePSUBankBridgeConnectionsGetFromConnectionID(
			infiniteRetryContext(ctx),
			handleWebhooks.ConnectorID,
			*response.DataReadyToFetch.ID,
		)
		if err != nil {
			return fmt.Errorf("getting bank bridge connection: %w", err)
		}

		bankBridge, err := activities.StoragePSUBankBridgesGet(
			infiniteRetryContext(ctx),
			psu,
			handleWebhooks.ConnectorID,
		)
		if err != nil {
			return fmt.Errorf("getting bank bridge: %w", err)
		}

		conn = connection
		ba = bankBridge
		connectionID = connection.ConnectionID
		psuID = psu
	}

	payload, err := json.Marshal(&models.BankBridgeFromPayload{
		PSUBankBridge:           ba,
		PSUBankBridgeConnection: conn,
		FromPayload:             response.DataReadyToFetch.FromPayload,
	})
	if err != nil {
		return fmt.Errorf("marshalling bank bridge from payload: %w", err)
	}

	config := models.DefaultConfig()
	if err := json.Unmarshal(connector.Config, &config); err != nil {
		return fmt.Errorf("unmarshalling connector config: %w", err)
	}

	fromPayload := &FromPayload{
		ID: func() string {
			if response.DataReadyToFetch.ID != nil {
				return *response.DataReadyToFetch.ID
			}

			return ""
		}(),
		Payload: payload,
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
		RunFetchBankBridgeData,
		FetchBankBridgeData{
			PsuID:        psuID,
			ConnectionID: connectionID,
			ConnectorID:  handleWebhooks.ConnectorID,
			Config:       config,
			FromPayload:  fromPayload,
		},
		[]models.ConnectorTaskTree{},
	).GetChildWorkflowExecution().Get(ctx, nil); err != nil {
		return fmt.Errorf("running transaction ready to fetch: %w", err)
	}

	return nil
}

func (w Workflow) handleUserLinkSessionFinishedWebhook(
	ctx workflow.Context,
	response models.WebhookResponse,
) error {
	attempt, err := activities.StoragePSUBankBridgeConnectionAttemptsGet(
		infiniteRetryContext(ctx),
		response.UserLinkSessionFinished.AttemptID,
	)
	if err != nil {
		return fmt.Errorf("getting bank bridge connection attempt: %w", err)
	}

	err = activities.StoragePSUBankBridgeConnectionAttemptsUpdateStatus(
		infiniteRetryContext(ctx),
		response.UserLinkSessionFinished.AttemptID,
		response.UserLinkSessionFinished.Status,
		response.UserLinkSessionFinished.Error,
	)
	if err != nil {
		return fmt.Errorf("updating bank bridge connection attempt status: %w", err)
	}

	sendEvent := SendEvents{
		UserLinkStatus: &models.UserLinkSessionFinished{
			PsuID:       attempt.PsuID,
			ConnectorID: attempt.ConnectorID,
			AttemptID:   attempt.ID,
			Status:      response.UserLinkSessionFinished.Status,
			Error:       response.UserLinkSessionFinished.Error,
		},
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
		RunSendEvents,
		sendEvent,
	).Get(ctx, nil); err != nil {
		return fmt.Errorf("sending events: %w", err)
	}

	return nil
}

func (w Workflow) handleUserPendingDisconnectWebhook(
	ctx workflow.Context,
	handleWebhooks HandleWebhooks,
	response models.WebhookResponse,
) error {
	_, psuID, err := activities.StoragePSUBankBridgeConnectionsGetFromConnectionID(
		infiniteRetryContext(ctx),
		handleWebhooks.ConnectorID,
		response.UserConnectionPendingDisconnect.ConnectionID,
	)
	if err != nil {
		return fmt.Errorf("getting bank bridge connection: %w", err)
	}

	sendEvent := SendEvents{
		UserPendingDisconnect: &models.UserConnectionPendingDisconnect{
			PsuID:        psuID,
			ConnectorID:  handleWebhooks.ConnectorID,
			ConnectionID: response.UserConnectionPendingDisconnect.ConnectionID,
			At:           response.UserConnectionPendingDisconnect.At,
			Reason:       response.UserConnectionPendingDisconnect.Reason,
		},
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
		RunSendEvents,
		sendEvent,
	).Get(ctx, nil); err != nil {
		return fmt.Errorf("sending events: %w", err)
	}

	return nil
}

func (w Workflow) handleUserDisconnectedWebhook(
	ctx workflow.Context,
	handleWebhooks HandleWebhooks,
	response models.WebhookResponse,
) error {
	connection, psuID, err := activities.StoragePSUBankBridgeConnectionsGetFromConnectionID(
		infiniteRetryContext(ctx),
		handleWebhooks.ConnectorID,
		response.UserConnectionDisconnected.ConnectionID,
	)
	if err != nil {
		return fmt.Errorf("getting bank bridge connection: %w", err)
	}

	connection.Status = models.ConnectionStatusError
	connection.Error = response.UserConnectionDisconnected.Reason

	err = activities.StoragePSUBankBridgeConnectionsStore(
		infiniteRetryContext(ctx),
		psuID,
		*connection,
	)
	if err != nil {
		return fmt.Errorf("storing bank bridge connection: %w", err)
	}

	sendEvent := SendEvents{
		UserDisconnected: &models.UserConnectionDisconnected{
			PsuID:        psuID,
			ConnectorID:  handleWebhooks.ConnectorID,
			ConnectionID: response.UserConnectionDisconnected.ConnectionID,
			At:           response.UserConnectionDisconnected.At,
			Reason:       response.UserConnectionDisconnected.Reason,
		},
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
		RunSendEvents,
		sendEvent,
	).Get(ctx, nil); err != nil {
		return fmt.Errorf("sending events: %w", err)
	}

	return nil
}

const RunHandleWebhooks = "RunHandleWebhooks"

type StoreWebhookTranslation struct {
	ConnectorID     models.ConnectorID
	Account         *models.PSPAccount
	ExternalAccount *models.PSPAccount
	Payment         *models.PSPPayment
}

func (w Workflow) runStoreWebhookTranslation(
	ctx workflow.Context,
	storeWebhookTranslation StoreWebhookTranslation,
) error {
	var sendEvent *SendEvents
	if storeWebhookTranslation.Account != nil {
		accounts, err := models.FromPSPAccounts(
			[]models.PSPAccount{*storeWebhookTranslation.Account},
			models.ACCOUNT_TYPE_INTERNAL,
			storeWebhookTranslation.ConnectorID,
			nil,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate accounts",
				ErrValidation,
				err,
			)
		}

		err = activities.StorageAccountsStore(
			infiniteRetryContext(ctx),
			accounts,
		)
		if err != nil {
			return fmt.Errorf("storing next accounts: %w", err)
		}

		sendEvent = &SendEvents{
			Account: pointer.For(accounts[0]),
		}
	}

	if storeWebhookTranslation.ExternalAccount != nil {
		accounts, err := models.FromPSPAccounts(
			[]models.PSPAccount{*storeWebhookTranslation.ExternalAccount},
			models.ACCOUNT_TYPE_EXTERNAL,
			storeWebhookTranslation.ConnectorID,
			nil,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate accounts",
				ErrValidation,
				err,
			)
		}

		err = activities.StorageAccountsStore(
			infiniteRetryContext(ctx),
			accounts,
		)
		if err != nil {
			return fmt.Errorf("storing next accounts: %w", err)
		}

		sendEvent = &SendEvents{
			Account: pointer.For(accounts[0]),
		}
	}

	if storeWebhookTranslation.Payment != nil {
		payments, err := models.FromPSPPayments(
			[]models.PSPPayment{*storeWebhookTranslation.Payment},
			storeWebhookTranslation.ConnectorID,
			nil,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate psp payments",
				ErrValidation,
				err,
			)
		}

		err = activities.StoragePaymentsStore(
			infiniteRetryContext(ctx),
			payments,
		)
		if err != nil {
			return fmt.Errorf("storing next payments: %w", err)
		}

		sendEvent = &SendEvents{
			Payment: pointer.For(payments[0]),
		}
	}

	if sendEvent != nil {
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
			*sendEvent,
		).Get(ctx, nil); err != nil {
			return fmt.Errorf("sending events: %w", err)
		}
	}

	return nil
}

const RunStoreWebhookTranslation = "RunStoreWebhookTranslation"
