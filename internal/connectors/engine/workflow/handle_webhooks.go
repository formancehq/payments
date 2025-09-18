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
				return fmt.Errorf("handling open banking webhook: %w", err)
			}

		case response.UserLinkSessionFinished != nil:
			// OpenBanking specific webhook. A user has finished the link flow
			// and has a valid connection to his bank. We need to update the
			// open banking status to active and send an event to the user.
			if err := w.handleUserLinkSessionFinishedWebhook(ctx, response); err != nil {
				return fmt.Errorf("handling user link session finished webhook: %w", err)
			}

		case response.UserDisconnected != nil:
			// OpenBanking specific webhook. A user has disconnected was totally
			// disconnected from the open banking connector. We need to update
			// the open banking status to disconnected and send an event to the
			// user.
			if err := w.handleUserDisconnectedWebhook(ctx, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling user disconnected webhook: %w", err)
			}

		case response.UserConnectionDisconnected != nil:
			// OpenBanking specific webhook. A user has disconnected from his
			// bank. We need to update the open banking status to disconnected
			// and send an event to the user.
			if err := w.handleUserConnectionDisconnectedWebhook(ctx, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling user disconnected webhook: %w", err)
			}

		case response.UserConnectionReconnected != nil:
			// OpenBanking specific webhook. A user has reconnected to his bank.
			// We need to update the open banking status to active and send an
			// event to the user.
			if err := w.handleUserConnectionReconnectedWebhook(ctx, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling user reconnected webhook: %w", err)
			}

		case response.UserConnectionPendingDisconnect != nil:
			// OpenBanking specific webhook. A user is nearly disconnected from
			// his bank. We need to send an event to the user to warn him.
			if err := w.handleUserPendingDisconnectWebhook(ctx, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling user pending disconnect webhook: %w", err)
			}

		case response.OpenBankingAccount != nil:
			// OpenBanking specific webhook. A new account has been found in the
			// bank. We need to store the account in the database.
			if err := w.handleOpenBankingAccountWebhook(ctx, i, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling open banking account webhook: %w", err)
			}

		case response.OpenBankingPayment != nil:
			// OpenBanking specific webhook. A new payment has been found in the
			// bank. We need to store the payment in the database.
			if err := w.handleOpenBankingPaymentWebhook(ctx, i, handleWebhooks, response); err != nil {
				return fmt.Errorf("handling open banking payment webhook: %w", err)
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
			Balance:         response.Balance,
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

func (w Workflow) handleOpenBankingAccountWebhook(
	ctx workflow.Context,
	index int,
	handleWebhooks HandleWebhooks,
	response models.WebhookResponse,
) error {
	account := models.PSPAccount{
		Reference:    response.OpenBankingAccount.Reference,
		CreatedAt:    response.OpenBankingAccount.CreatedAt,
		Name:         response.OpenBankingAccount.Name,
		DefaultAsset: response.OpenBankingAccount.DefaultAsset,
		Metadata:     response.OpenBankingAccount.Metadata,
		Raw:          response.OpenBankingAccount.Raw,
	}

	if account.Metadata == nil {
		account.Metadata = make(map[string]string)
	}

	if response.OpenBankingAccount.OpenBankingUserID != nil {
		forwardedUser, err := activities.StorageOpenBankingForwardedUsersGetByPSPUserID(
			infiniteRetryContext(ctx),
			*response.OpenBankingAccount.OpenBankingUserID,
			handleWebhooks.ConnectorID,
		)
		if err != nil {
			return fmt.Errorf("getting open banking forwarded user: %w", err)
		}

		account.PsuID = &forwardedUser.PsuID
	}

	if response.OpenBankingAccount.OpenBankingConnectionID != nil {
		account.OpenBankingConnectionID = response.OpenBankingAccount.OpenBankingConnectionID
	}

	return w.handleDataToStoreWebhook(ctx, index, handleWebhooks, models.WebhookResponse{
		Account: &account,
	})
}

func (w Workflow) handleOpenBankingPaymentWebhook(
	ctx workflow.Context,
	index int,
	handleWebhooks HandleWebhooks,
	response models.WebhookResponse,
) error {
	payment := models.PSPPayment{
		ParentReference:             response.OpenBankingPayment.ParentReference,
		Reference:                   response.OpenBankingPayment.Reference,
		CreatedAt:                   response.OpenBankingPayment.CreatedAt,
		Type:                        response.OpenBankingPayment.Type,
		Amount:                      response.OpenBankingPayment.Amount,
		Asset:                       response.OpenBankingPayment.Asset,
		Scheme:                      response.OpenBankingPayment.Scheme,
		Status:                      response.OpenBankingPayment.Status,
		SourceAccountReference:      response.OpenBankingPayment.SourceAccountReference,
		DestinationAccountReference: response.OpenBankingPayment.DestinationAccountReference,
		Metadata:                    response.OpenBankingPayment.Metadata,
		Raw:                         response.OpenBankingPayment.Raw,
	}

	if payment.Metadata == nil {
		payment.Metadata = make(map[string]string)
	}

	if response.OpenBankingPayment.OpenBankingUserID != nil {
		forwardedUser, err := activities.StorageOpenBankingForwardedUsersGetByPSPUserID(
			infiniteRetryContext(ctx),
			*response.OpenBankingPayment.OpenBankingUserID,
			handleWebhooks.ConnectorID,
		)
		if err != nil {
			return fmt.Errorf("getting open banking forwardedUser: %w", err)
		}

		payment.PsuID = &forwardedUser.PsuID
	}

	if response.OpenBankingPayment.OpenBankingConnectionID != nil {
		payment.OpenBankingConnectionID = response.OpenBankingPayment.OpenBankingConnectionID
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

	var conn *models.OpenBankingConnection
	var obForwardedUser *models.OpenBankingForwardedUser
	var psuID uuid.UUID
	var connectionID string
	if response.DataReadyToFetch.PSUID != nil {
		openBankingForwardedUser, err := activities.StorageOpenBankingForwardedUsersGet(
			infiniteRetryContext(ctx),
			*response.DataReadyToFetch.PSUID,
			handleWebhooks.ConnectorID,
		)
		if err != nil {
			return fmt.Errorf("getting open banking: %w", err)
		}

		obForwardedUser = openBankingForwardedUser
		psuID = obForwardedUser.PsuID
	}

	if response.DataReadyToFetch.ConnectionID != nil {
		connection, psu, err := activities.StorageOpenBankingConnectionsGetFromConnectionID(
			infiniteRetryContext(ctx),
			handleWebhooks.ConnectorID,
			*response.DataReadyToFetch.ConnectionID,
		)
		if err != nil {
			return fmt.Errorf("getting open banking connection: %w", err)
		}

		conn = connection
		connectionID = connection.ConnectionID
		psuID = psu
	}

	payload, err := json.Marshal(&models.OpenBankingForwardedUserFromPayload{
		PSUID:                    psuID,
		OpenBankingForwardedUser: obForwardedUser,
		OpenBankingConnection:    conn,
		FromPayload:              response.DataReadyToFetch.FromPayload,
	})
	if err != nil {
		return fmt.Errorf("marshalling open banking from payload: %w", err)
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
		RunFetchOpenBankingData,
		FetchOpenBankingData{
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
	attempt, err := activities.StorageOpenBankingConnectionAttemptsGet(
		infiniteRetryContext(ctx),
		response.UserLinkSessionFinished.AttemptID,
	)
	if err != nil {
		return fmt.Errorf("getting open banking connection attempt: %w", err)
	}

	err = activities.StorageOpenBankingConnectionAttemptsUpdateStatus(
		infiniteRetryContext(ctx),
		response.UserLinkSessionFinished.AttemptID,
		response.UserLinkSessionFinished.Status,
		response.UserLinkSessionFinished.Error,
	)
	if err != nil {
		return fmt.Errorf("updating open banking connection attempt status: %w", err)
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
	_, psuID, err := activities.StorageOpenBankingConnectionsGetFromConnectionID(
		infiniteRetryContext(ctx),
		handleWebhooks.ConnectorID,
		response.UserConnectionPendingDisconnect.ConnectionID,
	)
	if err != nil {
		return fmt.Errorf("getting open banking connection: %w", err)
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
	openBanking, err := activities.StorageOpenBankingForwardedUsersGetByPSPUserID(
		infiniteRetryContext(ctx),
		response.UserDisconnected.PSPUserID,
		handleWebhooks.ConnectorID,
	)
	if err != nil {
		return fmt.Errorf("getting open banking: %w", err)
	}

	sendEvent := SendEvents{
		UserDisconnected: &models.UserDisconnected{
			PsuID:       openBanking.PsuID,
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
	connection, psuID, err := activities.StorageOpenBankingConnectionsGetFromConnectionID(
		infiniteRetryContext(ctx),
		handleWebhooks.ConnectorID,
		response.UserConnectionDisconnected.ConnectionID,
	)
	if err != nil {
		if response.UserConnectionDisconnected.PSPUserID == "" {
			// Nothing more to do, we're missing crucial information in order to continue
			return fmt.Errorf("getting open banking connection: %w", err)
		}

		// Let's try to fetch the psu via the forwarded user
		user, errGetUser := activities.StorageOpenBankingForwardedUsersGetByPSPUserID(
			infiniteRetryContext(ctx),
			response.UserConnectionDisconnected.PSPUserID,
			handleWebhooks.ConnectorID,
		)
		if errGetUser != nil {
			return fmt.Errorf("error getting connection: %w and getting forwarded user by pspuserID: %w", err, errGetUser)
		}

		psuID = user.PsuID
	}

	updatedConnection := craftUpdatedConnection(
		ctx,
		response.UserConnectionDisconnected.ConnectionID,
		handleWebhooks.ConnectorID,
		connection,
		models.ConnectionStatusError,
		response.UserConnectionDisconnected.Reason,
	)

	err = activities.StorageOpenBankingConnectionsStore(
		infiniteRetryContext(ctx),
		psuID,
		updatedConnection,
	)
	if err != nil {
		return fmt.Errorf("storing open banking connection: %w", err)
	}

	sendEvent := SendEvents{
		UserConnectionDisconnected: &models.UserConnectionDisconnected{
			PsuID:        psuID,
			ConnectorID:  handleWebhooks.ConnectorID,
			ConnectionID: response.UserConnectionDisconnected.ConnectionID,
			ErrorType:    response.UserConnectionDisconnected.ErrorType,
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
	connection, psuID, err := activities.StorageOpenBankingConnectionsGetFromConnectionID(
		infiniteRetryContext(ctx),
		handleWebhooks.ConnectorID,
		response.UserConnectionReconnected.ConnectionID,
	)
	if err != nil {
		if response.UserConnectionReconnected.PSPUserID == "" {
			// Nothing more to do, we're missing crucial information in order to continue
			return fmt.Errorf("getting open banking connection: %w", err)
		}

		// Let's try to fetch the ob forwarded user
		user, errGetUser := activities.StorageOpenBankingForwardedUsersGetByPSPUserID(
			infiniteRetryContext(ctx),
			response.UserConnectionReconnected.PSPUserID,
			handleWebhooks.ConnectorID,
		)
		if errGetUser != nil {
			return fmt.Errorf("error getting connection: %w and getting forwarded user by psuId: %w", err, errGetUser)
		}

		psuID = user.PsuID
	}

	updatedConnection := craftUpdatedConnection(
		ctx,
		response.UserConnectionReconnected.ConnectionID,
		handleWebhooks.ConnectorID,
		connection,
		models.ConnectionStatusActive,
		nil,
	)

	err = activities.StorageOpenBankingConnectionsStore(
		infiniteRetryContext(ctx),
		psuID,
		updatedConnection,
	)
	if err != nil {
		return fmt.Errorf("storing open banking connection: %w", err)
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
	Balance         *models.PSPBalance
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

	if storeWebhookTranslation.Balance != nil {
		var psuId *uuid.UUID
		var openBankingConnectionID *string

		acc, err := activities.StorageAccountsGet(
			infiniteRetryContext(ctx),
			models.AccountID{
				Reference:   storeWebhookTranslation.Balance.AccountReference,
				ConnectorID: storeWebhookTranslation.ConnectorID,
			},
		)
		// there might be cases where the account is not found, should we accept and continue?
		if err != nil && acc != nil {
			psuId = acc.PsuID
			openBankingConnectionID = acc.OpenBankingConnectionID
		}

		balance, err := models.FromPSPBalance(
			*storeWebhookTranslation.Balance,
			storeWebhookTranslation.ConnectorID,
			psuId,
			openBankingConnectionID,
		)
		if err != nil {
			return temporal.NewNonRetryableApplicationError(
				"failed to translate balances",
				ErrValidation,
				err,
			)
		}

		err = activities.StorageBalancesStore(
			infiniteRetryContext(ctx),
			[]models.Balance{balance},
		)
		if err != nil {
			return fmt.Errorf("storing next balances: %w", err)
		}

		sendEvent = &SendEvents{
			Balance: pointer.For(balance),
		}
	}

	if storeWebhookTranslation.ExternalAccount != nil {
		accounts, err := models.FromPSPAccounts(
			[]models.PSPAccount{*storeWebhookTranslation.ExternalAccount},
			models.ACCOUNT_TYPE_EXTERNAL,
			storeWebhookTranslation.ConnectorID,
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
