package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type ReverseTransfer struct {
	TaskID                      models.TaskID
	ConnectorID                 models.ConnectorID
	PaymentInitiationReversalID models.PaymentInitiationReversalID
}

func (w Workflow) runReverseTransfer(
	ctx workflow.Context,
	reverseTransfer ReverseTransfer,
) error {
	paymentID, err := w.reverseTransfer(ctx, reverseTransfer)
	if err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			reverseTransfer.TaskID,
			reverseTransfer.ConnectorID,
			err,
		)
		if errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return w.updateTaskSuccess(
		ctx,
		reverseTransfer.TaskID,
		reverseTransfer.ConnectorID,
		paymentID,
	)
}

func (w Workflow) reverseTransfer(
	ctx workflow.Context,
	reverseTransfer ReverseTransfer,
) (string, error) {
	// Get the payment initiation reversal
	piReversal, err := activities.StoragePaymentInitiationReversalsGet(
		infiniteRetryContext(ctx),
		reverseTransfer.PaymentInitiationReversalID,
	)
	if err != nil {
		return "", err
	}

	pi, err := activities.StoragePaymentInitiationsGet(
		infiniteRetryContext(ctx),
		piReversal.PaymentInitiationID,
	)
	if err != nil {
		return "", err
	}

	if err := w.validateReverse(
		ctx,
		ValidateReverse{
			ConnectorID: reverseTransfer.ConnectorID,
			PI:          pi,
			PIReversal:  piReversal,
		},
	); err != nil {
		return "", err
	}

	pspPI, err := w.getPSPPI(ctx, pi)
	if err != nil {
		return "", err
	}

	pspReversal := models.FromPaymentInitiationReversalToPSPPaymentInitiationReversal(
		piReversal,
		pspPI,
	)

	reverseTransferResponse, errPlugin := activities.PluginReverseTransfer(
		infiniteRetryContext(ctx),
		reverseTransfer.ConnectorID,
		models.ReverseTransferRequest{
			PaymentInitiationReversal: pspReversal,
		},
	)
	switch errPlugin {
	case nil:
		payment := models.FromPSPPaymentToPayment(reverseTransferResponse.Payment, reverseTransfer.ConnectorID)

		// Store refund for the payment initiation
		if err := w.storePIPaymentWithStatus(
			ctx,
			payment,
			pi.ID,
			getPIStatusFromPayment(payment.Status),
			reverseTransfer.ConnectorID,
		); err != nil {
			return "", err
		}

		err := w.addPIReversalAdjustment(
			ctx,
			models.PaymentInitiationReversalAdjustmentID{
				PaymentInitiationReversalID: reverseTransfer.PaymentInitiationReversalID,
				CreatedAt:                   workflow.Now(ctx),
				Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_PROCESSED,
			},
			nil,
			nil,
		)
		if err != nil {
			return "", err
		}

		return payment.ID.String(), nil

	default:
		err := w.addPIReversalAdjustment(
			ctx,
			models.PaymentInitiationReversalAdjustmentID{
				PaymentInitiationReversalID: reverseTransfer.PaymentInitiationReversalID,
				CreatedAt:                   workflow.Now(ctx),
				Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_FAILED,
			},
			errPlugin,
			nil,
		)
		if err != nil {
			return "", err
		}

		return "", errPlugin
	}
}

var RunReverseTransfer = "ReverseTransfer"
