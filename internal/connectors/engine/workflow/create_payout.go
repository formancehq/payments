package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type CreatePayout struct {
	TaskID              models.TaskID
	ConnectorID         models.ConnectorID
	PaymentInitiationID models.PaymentInitiationID
}

func (w Workflow) runCreatePayout(
	ctx workflow.Context,
	createPayout CreatePayout,
) error {
	pID, err := w.createPayout(ctx, createPayout)
	if err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			createPayout.TaskID,
			createPayout.ConnectorID,
			err,
		)
		if errUpdateTask != nil {
			return errUpdateTask
		}
	}

	return w.updateTaskSucces(
		ctx,
		createPayout.TaskID,
		createPayout.ConnectorID,
		pID.String(),
	)
}

func (w Workflow) createPayout(
	ctx workflow.Context,
	createPayout CreatePayout,
) (models.PaymentID, error) {
	// Get the payment initiation
	pi, err := activities.StoragePaymentInitiationsGet(
		infiniteRetryContext(ctx),
		createPayout.PaymentInitiationID,
	)
	if err != nil {
		return models.PaymentID{}, err
	}

	var sourceAccount *models.Account
	if pi.SourceAccountID != nil {
		sourceAccount, err = activities.StorageAccountsGet(
			infiniteRetryContext(ctx),
			*pi.SourceAccountID,
		)
		if err != nil {
			return models.PaymentID{}, err
		}
	}

	var destinationAccount *models.Account
	if pi.DestinationAccountID != nil {
		destinationAccount, err = activities.StorageAccountsGet(
			infiniteRetryContext(ctx),
			*pi.DestinationAccountID,
		)
		if err != nil {
			return models.PaymentID{}, err
		}
	}

	pspPI := models.FromPaymentInitiationToPSPPaymentInitiation(pi, models.ToPSPAccount(sourceAccount), models.ToPSPAccount(destinationAccount))

	err = w.addPIAdjustment(
		ctx,
		models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: createPayout.PaymentInitiationID,
			CreatedAt:           workflow.Now(ctx),
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
		},
		nil,
		nil,
	)
	if err != nil {
		return models.PaymentID{}, err
	}

	createPayoutResponse, errPlugin := activities.PluginCreatePayout(
		infiniteRetryContext(ctx),
		createPayout.ConnectorID,
		models.CreatePayoutRequest{
			PaymentInitiation: *pspPI,
		},
	)
	switch errPlugin {
	case nil:
		payment := models.FromPSPPaymentToPayment(createPayoutResponse.Payment, createPayout.ConnectorID)

		err = activities.StoragePaymentsStore(
			infiniteRetryContext(ctx),
			[]models.Payment{payment},
		)
		if err != nil {
			return models.PaymentID{}, err
		}

		err = activities.StoragePaymentInitiationsRelatedPaymentsStore(
			infiniteRetryContext(ctx),
			createPayout.PaymentInitiationID,
			payment.ID,
			createPayoutResponse.Payment.CreatedAt,
		)
		if err != nil {
			return models.PaymentID{}, err
		}

		err = w.addPIAdjustment(
			ctx,
			models.PaymentInitiationAdjustmentID{
				PaymentInitiationID: createPayout.PaymentInitiationID,
				CreatedAt:           workflow.Now(ctx),
				Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
			},
			nil,
			nil,
		)
		if err != nil {
			return models.PaymentID{}, err
		}

		return payment.ID, nil
	default:
		err = w.addPIAdjustment(
			ctx,
			models.PaymentInitiationAdjustmentID{
				PaymentInitiationID: createPayout.PaymentInitiationID,
				CreatedAt:           workflow.Now(ctx),
				Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
			},
			err,
			nil,
		)
		if err != nil {
			return models.PaymentID{}, err
		}

		return models.PaymentID{}, errPlugin
	}
}

const RunCreatePayout = "CreatePayout"
