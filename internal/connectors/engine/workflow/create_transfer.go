package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type CreateTransfer struct {
	ConnectorID         models.ConnectorID
	PaymentInitiationID models.PaymentInitiationID
}

func (w Workflow) runCreateTransfer(
	ctx workflow.Context,
	createTransfer CreateTransfer,
) error {
	// Get the payment initiation
	pi, err := activities.StoragePaymentInitiationsGet(
		infiniteRetryContext(ctx),
		createTransfer.PaymentInitiationID,
	)
	if err != nil {
		return err
	}

	var sourceAccount *models.Account
	if pi.SourceAccountID != nil {
		sourceAccount, err = activities.StorageAccountsGet(
			infiniteRetryContext(ctx),
			*pi.SourceAccountID,
		)
		if err != nil {
			return err
		}
	}

	var destinationAccount *models.Account
	if pi.DestinationAccountID != nil {
		destinationAccount, err = activities.StorageAccountsGet(
			infiniteRetryContext(ctx),
			*pi.DestinationAccountID,
		)
		if err != nil {
			return err
		}
	}

	pspPI := models.FromPaymentInitiationToPSPPaymentInitiation(pi, models.ToPSPAccount(sourceAccount), models.ToPSPAccount(destinationAccount))

	err = w.addPIAdjustment(
		ctx,
		models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: createTransfer.PaymentInitiationID,
			CreatedAt:           workflow.Now(ctx),
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
		},
		nil,
		nil,
	)
	if err != nil {
		return err
	}

	createTransferResponse, errPlugin := activities.PluginCreateTransfer(
		infiniteRetryContext(ctx),
		createTransfer.ConnectorID,
		models.CreateTransferRequest{
			PaymentInitiation: *pspPI,
		},
	)
	switch errPlugin {
	case nil:
		payment := models.FromPSPPaymentToPayment(createTransferResponse.Payment, createTransfer.ConnectorID)

		err := activities.StoragePaymentsStore(
			infiniteRetryContext(ctx),
			[]models.Payment{payment},
		)
		if err != nil {
			return errPlugin
		}

		err = activities.StoragePaymentInitiationsRelatedPaymentsStore(
			infiniteRetryContext(ctx),
			createTransfer.PaymentInitiationID,
			payment.ID,
			createTransferResponse.Payment.CreatedAt,
		)
		if err != nil {
			return errPlugin
		}

		err = w.addPIAdjustment(
			ctx,
			models.PaymentInitiationAdjustmentID{
				PaymentInitiationID: createTransfer.PaymentInitiationID,
				CreatedAt:           workflow.Now(ctx),
				Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
			},
			nil,
			nil,
		)
		if err != nil {
			return errPlugin
		}

		return nil
	default:
		err := w.addPIAdjustment(
			ctx,
			models.PaymentInitiationAdjustmentID{
				PaymentInitiationID: createTransfer.PaymentInitiationID,
				CreatedAt:           workflow.Now(ctx),
				Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
			},
			errPlugin,
			nil,
		)
		if err != nil {
			return err
		}

		return errPlugin
	}
}

var RunCreateTransfer any

func init() {
	RunCreateTransfer = Workflow{}.runCreateTransfer
}

func (w Workflow) addPIAdjustment(
	ctx workflow.Context,
	adjustmentID models.PaymentInitiationAdjustmentID,
	err error,
	metadata map[string]string,
) error {
	adj := models.PaymentInitiationAdjustment{
		ID:                  adjustmentID,
		PaymentInitiationID: adjustmentID.PaymentInitiationID,
		CreatedAt:           workflow.Now(ctx),
		Status:              adjustmentID.Status,
		Error:               err,
		Metadata:            metadata,
	}

	return activities.StoragePaymentInitiationsAdjustmentsStore(
		infiniteRetryContext(ctx),
		adj,
	)
}
