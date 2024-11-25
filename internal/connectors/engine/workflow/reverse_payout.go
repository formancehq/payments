package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type ReversePayout struct {
	TaskID                      models.TaskID
	ConnectorID                 models.ConnectorID
	PaymentInitiationReversalID models.PaymentInitiationReversalID
}

func (w Workflow) runReversePayout(
	ctx workflow.Context,
	reversePayout ReversePayout,
) error {
	paymentID, err := w.reversePayout(ctx, reversePayout)
	if err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			reversePayout.TaskID,
			reversePayout.ConnectorID,
			err,
		)
		if errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return w.updateTaskSuccess(
		ctx,
		reversePayout.TaskID,
		reversePayout.ConnectorID,
		paymentID,
	)
}

func (w Workflow) reversePayout(
	ctx workflow.Context,
	reversePayout ReversePayout,
) (string, error) {
	// Get the payment initiation reversal
	piReversal, err := activities.StoragePaymentInitiationReversalsGet(
		infiniteRetryContext(ctx),
		reversePayout.PaymentInitiationReversalID,
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
			ConnectorID: reversePayout.ConnectorID,
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

	reversePayoutResponse, errPlugin := activities.PluginReversePayout(
		infiniteRetryContext(ctx),
		reversePayout.ConnectorID,
		models.ReversePayoutRequest{
			PaymentInitiationReversal: pspReversal,
		},
	)
	switch errPlugin {
	case nil:
		payment := models.FromPSPPaymentToPayment(reversePayoutResponse.Payment, reversePayout.ConnectorID)

		// Store refund for the payment initiation
		if err := w.storePIPaymentWithStatus(
			ctx,
			payment,
			pi.ID,
			getPIStatusFromPayment(payment.Status),
			reversePayout.ConnectorID,
		); err != nil {
			return "", err
		}

		err := w.addPIReversalAdjustment(
			ctx,
			models.PaymentInitiationReversalAdjustmentID{
				PaymentInitiationReversalID: reversePayout.PaymentInitiationReversalID,
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
				PaymentInitiationReversalID: reversePayout.PaymentInitiationReversalID,
				CreatedAt:                   workflow.Now(ctx),
				Status:                      models.PAYMENT_INITIATION_REVERSAL_STATUS_FAILED,
			},
			errPlugin,
			nil,
		)
		if err != nil {
			return "", err
		}

		err = w.addPIAdjustment(
			ctx,
			models.PaymentInitiationAdjustmentID{
				PaymentInitiationID: pi.ID,
				CreatedAt:           workflow.Now(ctx),
				Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED,
			},
			pi.Amount,
			&pi.Asset,
			nil,
			nil,
		)
		if err != nil {
			return "", err
		}

		return "", errPlugin
	}
}

var RunReversePayout = "ReversePayout"
