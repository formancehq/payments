package workflow

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

func (w Workflow) storePIPaymentWithStatus(
	ctx workflow.Context,
	payment models.Payment,
	paymentInitiationID models.PaymentInitiationID,
	status models.PaymentInitiationAdjustmentStatus,
) error {
	// payment is available, storing it
	err := activities.StoragePaymentsStore(
		infiniteRetryContext(ctx),
		[]models.Payment{payment},
	)
	if err != nil {
		return err
	}

	err = activities.StoragePaymentInitiationsRelatedPaymentsStore(
		infiniteRetryContext(ctx),
		paymentInitiationID,
		payment.ID,
		payment.CreatedAt,
	)
	if err != nil {
		return err
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
		SendEvents{
			Payment: &payment,
			PaymentInitiationRelatedPayment: &models.PaymentInitiationRelatedPayments{
				PaymentInitiationID: paymentInitiationID,
				PaymentID:           payment.ID,
			},
		},
	).Get(ctx, nil); err != nil {
		return err
	}

	err = w.addPIAdjustment(
		ctx,
		models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: paymentInitiationID,
			CreatedAt:           workflow.Now(ctx),
			Status:              status,
		},
		payment.Amount,
		&payment.Asset,
		nil,
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (w Workflow) addPIAdjustment(
	ctx workflow.Context,
	adjustmentID models.PaymentInitiationAdjustmentID,
	amount *big.Int,
	asset *string,
	err error,
	metadata map[string]string,
) error {
	adj := models.PaymentInitiationAdjustment{
		ID:        adjustmentID,
		CreatedAt: workflow.Now(ctx),
		Status:    adjustmentID.Status,
		Amount:    amount,
		Asset:     asset,
		Error:     err,
		Metadata:  metadata,
	}

	if err := activities.StoragePaymentInitiationsAdjustmentsStore(
		infiniteRetryContext(ctx),
		adj,
	); err != nil {
		return err
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
		SendEvents{
			PaymentInitiationAdjustment: &adj,
		},
	).Get(ctx, nil); err != nil {
		return err
	}

	return nil
}

func (w Workflow) getPSPPI(
	ctx workflow.Context,
	pi *models.PaymentInitiation,
) (models.PSPPaymentInitiation, error) {
	var sourceAccount *models.Account
	if pi.SourceAccountID != nil {
		var err error
		sourceAccount, err = activities.StorageAccountsGet(
			infiniteRetryContext(ctx),
			*pi.SourceAccountID,
		)
		if err != nil {
			return models.PSPPaymentInitiation{}, err
		}
	}

	var destinationAccount *models.Account
	if pi.DestinationAccountID != nil {
		var err error
		destinationAccount, err = activities.StorageAccountsGet(
			infiniteRetryContext(ctx),
			*pi.DestinationAccountID,
		)
		if err != nil {
			return models.PSPPaymentInitiation{}, err
		}

		if destinationAccount.Type == models.ACCOUNT_TYPE_EXTERNAL {
			if err := fillFormanceBankAccount(ctx, destinationAccount); err != nil {
				return models.PSPPaymentInitiation{}, err
			}
		}
	}
	pspPI := models.FromPaymentInitiationToPSPPaymentInitiation(pi, models.ToPSPAccount(sourceAccount), models.ToPSPAccount(destinationAccount))
	return pspPI, nil
}

func fillFormanceBankAccount(
	ctx workflow.Context,
	account *models.Account,
) error {
	bankAccountUUID, err := uuid.Parse(account.ID.Reference)
	if err != nil {
		// Not an uuid, so cannot be a formance bank account
		return nil
	}

	bankAccount, err := activities.StorageBankAccountsGet(
		infiniteRetryContext(ctx),
		bankAccountUUID,
		true,
	)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		}
		return err
	}

	models.FillBankAccountDetailsToAccountMetadata(account, bankAccount)

	return nil
}

func getPIStatusFromPayment(status models.PaymentStatus) models.PaymentInitiationAdjustmentStatus {
	switch status {
	case models.PAYMENT_STATUS_SUCCEEDED,
		models.PAYMENT_STATUS_CAPTURE,
		models.PAYMENT_STATUS_REFUND_REVERSED:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED
	case models.PAYMENT_STATUS_CANCELLED,
		models.PAYMENT_STATUS_CAPTURE_FAILED,
		models.PAYMENT_STATUS_FAILED,
		models.PAYMENT_STATUS_EXPIRED:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED
	case models.PAYMENT_STATUS_PENDING,
		models.PAYMENT_STATUS_AUTHORISATION:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING
	case models.PAYMENT_STATUS_REFUNDED:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED
	case models.PAYMENT_STATUS_REFUNDED_FAILURE:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED
	default:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN
	}
}

func craftUpdatedConnection(
	ctx workflow.Context,
	connectionID string,
	connectorID models.ConnectorID,
	connection *models.PSUBankBridgeConnection,
	updatedStatus models.ConnectionStatus,
	updatedError *string,
) models.PSUBankBridgeConnection {
	now := workflow.Now(ctx)
	updatedConnection := models.PSUBankBridgeConnection{
		ConnectionID: connectionID,
		ConnectorID:  connectorID,
		CreatedAt:    now,
		UpdatedAt:    now,
		Status:       updatedStatus,
		Error:        updatedError,
	}

	if connection != nil {
		updatedConnection.CreatedAt = connection.CreatedAt
		updatedConnection.AccessToken = connection.AccessToken
		updatedConnection.Metadata = connection.Metadata
		updatedConnection.DataUpdatedAt = connection.DataUpdatedAt
	}

	return updatedConnection
}

func (w Workflow) getDefaultTaskQueue() string {
	return fmt.Sprintf("%s-default", w.stack)
}
