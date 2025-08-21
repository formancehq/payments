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

	for i, response := range resp.Responses {
		switch {
		case response.DataReadyToFetch != nil:
			// A webhook has been received from the connector indicating that
			// there is new data to fetch from the connector.
			// Let's launch the related workflow to fetch the data.
			if err := w.handleTransactionReadyToFetchWebhook(ctx, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling bank bridge webhook: %w", err)
			}

		case response.UserLinkSessionFinished != nil:
			// BankBridge specific webhook. A user has finished the link flow
			// and has a valid connection to his bank. We need to update the
			// bank bridge status to active and send an event to the user.
			if err := w.handleUserLinkSessionFinishedWebhook(ctx, response); err != nil {
				return fmt.Errorf("handling user link session finished webhook: %w", err)
			}

		case response.UserDisconnected != nil:
			// BankBridge specific webhook. A user has disconnected was totally
			// disconnected from the bank bridge connector. We need to update
			// the bank bridge status to disconnected and send an event to the
			// user.
			if err := w.handleUserDisconnectedWebhook(ctx, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling user disconnected webhook: %w", err)
			}

		case response.UserConnectionDisconnected != nil:
			// BankBridge specific webhook. A user has disconnected from his
			// bank. We need to update the bank bridge status to disconnected
			// and send an event to the user.
			if err := w.handleUserConnectionDisconnectedWebhook(ctx, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling user disconnected webhook: %w", err)
			}

		case response.UserConnectionReconnected != nil:
			// BankBridge specific webhook. A user has reconnected to his bank.
			// We need to update the bank bridge status to active and send an
			// event to the user.
			if err := w.handleUserConnectionReconnectedWebhook(ctx, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling user reconnected webhook: %w", err)
			}

		case response.UserConnectionPendingDisconnect != nil:
			// BankBridge specific webhook. A user is nearly disconnected from
			// his bank. We need to send an event to the user to warn him.
			if err := w.handleUserPendingDisconnectWebhook(ctx, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling user pending disconnect webhook: %w", err)
			}

		case response.BankBridgeAccount != nil:
			// BankBridge specific webhook. A new account has been found in the
			// bank. We need to store the account in the database.
			if err := w.handleBankBridgeAccountWebhook(ctx, i, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling bank bridge account webhook: %w", err)
			}

		case response.BankBridgePayment != nil:
			// BankBridge specific webhook. A new payment has been found in the
			// bank. We need to store the payment in the database.
			if err := w.handleBankBridgePaymentWebhook(ctx, i, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling bank bridge payment webhook: %w", err)
			}

		default:
			// Default case, all the other webhooks are to store data
			if err := w.handleDataToStoreWebhook(ctx, i, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling data to store webhook: %w", err)
			}

		}
	}

	return nil
}

func (w Workflow) handleDataToStoreWebhook(
	ctx workflow.Context,
	index int,
	handleWebhooks HandleWebhooks,
	response models.WebhookResponse,
) error {
	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				WorkflowID:            fmt.Sprintf("store-webhook-%s-%s-%s-%d", w.stack, handleWebhooks.ConnectorID.String(), handleWebhooks.Webhook.ID, index),
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
			PaymentToDelete: response.PaymentToDelete,
			PaymentToCancel: response.PaymentToCancel,
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

func (w Workflow) handleBankBridgeAccountWebhook(
	ctx workflow.Context,
	index int,
	handleWebhooks HandleWebhooks,
	response models.WebhookResponse,
) error {
	account := models.PSPAccount{
		Reference:    response.BankBridgeAccount.Reference,
		CreatedAt:    response.BankBridgeAccount.CreatedAt,
		Name:         response.BankBridgeAccount.Name,
		DefaultAsset: response.BankBridgeAccount.DefaultAsset,
		Metadata:     response.BankBridgeAccount.Metadata,
		Raw:          response.BankBridgeAccount.Raw,
	}

	if account.Metadata == nil {
		account.Metadata = make(map[string]string)
	}

	if response.BankBridgeAccount.BankBridgeUserID != nil {
		bridge, err := activities.StoragePSUBankBridgesGetByPSPUserID(
			infiniteRetryContext(ctx),
			*response.BankBridgeAccount.BankBridgeUserID,
			handleWebhooks.ConnectorID,
		)
		if err != nil {
			return fmt.Errorf("getting bank bridge: %w", err)
		}

		account.Metadata[models.ObjectPSUIDMetadataKey] = bridge.PsuID.String()
	}

	if response.BankBridgeAccount.BankBridgeConnectionID != nil {
		account.Metadata[models.ObjectConnectionIDMetadataKey] = *response.BankBridgeAccount.BankBridgeConnectionID
	}

	return w.handleDataToStoreWebhook(ctx, index, handleWebhooks, models.WebhookResponse{
		Account: &account,
	})
}

func (w Workflow) handleBankBridgePaymentWebhook(
	ctx workflow.Context,
	index int,
	handleWebhooks HandleWebhooks,
	response models.WebhookResponse,
) error {
	payment := models.PSPPayment{
		ParentReference:             response.BankBridgePayment.ParentReference,
		Reference:                   response.BankBridgePayment.Reference,
		CreatedAt:                   response.BankBridgePayment.CreatedAt,
		Type:                        response.BankBridgePayment.Type,
		Amount:                      response.BankBridgePayment.Amount,
		Asset:                       response.BankBridgePayment.Asset,
		Scheme:                      response.BankBridgePayment.Scheme,
		Status:                      response.BankBridgePayment.Status,
		SourceAccountReference:      response.BankBridgePayment.SourceAccountReference,
		DestinationAccountReference: response.BankBridgePayment.DestinationAccountReference,
		Metadata:                    response.BankBridgePayment.Metadata,
		Raw:                         response.BankBridgePayment.Raw,
	}

	if payment.Metadata == nil {
		payment.Metadata = make(map[string]string)
	}

	if response.BankBridgePayment.BankBridgeUserID != nil {
		bridge, err := activities.StoragePSUBankBridgesGetByPSPUserID(
			infiniteRetryContext(ctx),
			*response.BankBridgePayment.BankBridgeUserID,
			handleWebhooks.ConnectorID,
		)
		if err != nil {
			return fmt.Errorf("getting bank bridge: %w", err)
		}

		payment.Metadata[models.ObjectPSUIDMetadataKey] = bridge.PsuID.String()
	}

	if response.BankBridgePayment.BankBridgeConnectionID != nil {
		payment.Metadata[models.ObjectConnectionIDMetadataKey] = *response.BankBridgePayment.BankBridgeConnectionID
	}

	return w.handleDataToStoreWebhook(ctx, index, handleWebhooks, models.WebhookResponse{
		Payment: &payment,
	})
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
	if response.DataReadyToFetch.PSUID != nil {
		bankBridge, err := activities.StoragePSUBankBridgesGet(
			infiniteRetryContext(ctx),
			*response.DataReadyToFetch.PSUID,
			handleWebhooks.ConnectorID,
		)
		if err != nil {
			return fmt.Errorf("getting bank bridge: %w", err)
		}

		ba = bankBridge
		psuID = bankBridge.PsuID
	}

	if response.DataReadyToFetch.ConnectionID != nil {
		connection, psu, err := activities.StoragePSUBankBridgeConnectionsGetFromConnectionID(
			infiniteRetryContext(ctx),
			handleWebhooks.ConnectorID,
			*response.DataReadyToFetch.ConnectionID,
		)
		if err != nil {
			return fmt.Errorf("getting bank bridge connection: %w", err)
		}

		conn = connection
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
			if response.DataReadyToFetch.ConnectionID != nil {
				return *response.DataReadyToFetch.ConnectionID
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
	bridge, err := activities.StoragePSUBankBridgesGetByPSPUserID(
		infiniteRetryContext(ctx),
		response.UserDisconnected.UserID,
		handleWebhooks.ConnectorID,
	)
	if err != nil {
		return fmt.Errorf("getting bank bridge: %w", err)
	}

	sendEvent := SendEvents{
		UserDisconnected: &models.UserDisconnected{
			PsuID:       bridge.PsuID,
			ConnectorID: handleWebhooks.ConnectorID,
			At:          workflow.Now(ctx),
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

func (w Workflow) handleUserConnectionDisconnectedWebhook(
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
		UserConnectionDisconnected: &models.UserConnectionDisconnected{
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

func (w Workflow) handleUserConnectionReconnectedWebhook(
	ctx workflow.Context,
	handleWebhooks HandleWebhooks,
	response models.WebhookResponse,
) error {
	connection, psuID, err := activities.StoragePSUBankBridgeConnectionsGetFromConnectionID(
		infiniteRetryContext(ctx),
		handleWebhooks.ConnectorID,
		response.UserConnectionReconnected.ConnectionID,
	)
	if err != nil {
		return fmt.Errorf("getting bank bridge connection: %w", err)
	}

	connection.Status = models.ConnectionStatusActive
	connection.Error = nil

	err = activities.StoragePSUBankBridgeConnectionsStore(
		infiniteRetryContext(ctx),
		psuID,
		*connection,
	)
	if err != nil {
		return fmt.Errorf("storing bank bridge connection: %w", err)
	}

	sendEvent := SendEvents{
		UserConnectionReconnected: &models.UserConnectionReconnected{
			PsuID:        psuID,
			ConnectorID:  handleWebhooks.ConnectorID,
			ConnectionID: response.UserConnectionReconnected.ConnectionID,
			At:           response.UserConnectionReconnected.At,
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
	PaymentToDelete *models.PSPPaymentsToDelete
	PaymentToCancel *models.PSPPaymentsToCancel
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

	if storeWebhookTranslation.PaymentToDelete != nil {
		payment, err := activities.StoragePaymentsGetByReference(
			infiniteRetryContext(ctx),
			storeWebhookTranslation.PaymentToDelete.Reference,
			storeWebhookTranslation.ConnectorID,
		)
		if err != nil {
			return fmt.Errorf("getting payment: %w", err)
		}

		err = activities.StoragePaymentsDeleteFromReference(
			infiniteRetryContext(ctx),
			storeWebhookTranslation.PaymentToDelete.Reference,
			storeWebhookTranslation.ConnectorID,
		)
		if err != nil {
			return fmt.Errorf("deleting payment: %w", err)
		}

		sendEvent = &SendEvents{
			PaymentDeleted: &payment.ID,
		}
	}

	if storeWebhookTranslation.PaymentToCancel != nil {
		payment, err := activities.StoragePaymentsGetByReference(
			infiniteRetryContext(ctx),
			storeWebhookTranslation.PaymentToCancel.Reference,
			storeWebhookTranslation.ConnectorID,
		)
		if err != nil {
			return fmt.Errorf("getting payment: %w", err)
		}

		now := workflow.Now(ctx)
		payment.Adjustments = []models.PaymentAdjustment{
			{
				ID: models.PaymentAdjustmentID{
					PaymentID: payment.ID,
					Reference: storeWebhookTranslation.PaymentToCancel.Reference,
					CreatedAt: now,
					Status:    models.PAYMENT_STATUS_CANCELLED,
				},
				Reference: storeWebhookTranslation.PaymentToCancel.Reference,
				CreatedAt: now,
				Status:    models.PAYMENT_STATUS_CANCELLED,
			},
		}

		err = activities.StoragePaymentsStore(
			infiniteRetryContext(ctx),
			[]models.Payment{*payment},
		)
		if err != nil {
			return fmt.Errorf("storing payment: %w", err)
		}

		sendEvent = &SendEvents{
			Payment: payment,
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
