package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type PollPayout struct {
	TaskID              models.TaskID
	ConnectorID         models.ConnectorID
	PaymentInitiationID models.PaymentInitiationID
	PayoutID            string
	ScheduleID          string
}

func (w Workflow) runPollPayout(
	ctx workflow.Context,
	pollPayout PollPayout,
) error {
	paymentID, err := w.pollPayout(ctx, pollPayout)
	if err != nil {
		return w.updateTasksError(
			ctx,
			pollPayout.TaskID,
			&pollPayout.ConnectorID,
			err,
		)
	}

	if paymentID != "" {
		return w.updateTaskSuccess(
			ctx,
			pollPayout.TaskID,
			&pollPayout.ConnectorID,
			paymentID,
		)
	}

	return nil
}

func (w Workflow) pollPayout(
	ctx workflow.Context,
	pollPayout PollPayout,
) (string, error) {
	pollPayoutStatusResponse, err := activities.PluginPollPayoutStatus(
		infiniteRetryContext(ctx),
		pollPayout.ConnectorID,
		models.PollPayoutStatusRequest{
			PayoutID: pollPayout.PayoutID,
		},
	)
	if err != nil {
		return "", err
	}

	paymentID := ""
	var piErr error
	isFinal := false

	switch {
	case pollPayoutStatusResponse.Payment == nil && pollPayoutStatusResponse.Error == nil:
		// payment not yet available and no error, waiting for the next polling
		return "", nil

	case pollPayoutStatusResponse.Payment != nil:
		payment, err := models.FromPSPPaymentToPayment(*pollPayoutStatusResponse.Payment, pollPayout.ConnectorID)
		if err != nil {
			return "", temporal.NewNonRetryableApplicationError(
				"failed to translate psp payment",
				ErrValidation,
				err,
			)
		}

		if err := w.storePIPaymentWithStatus(
			ctx,
			payment,
			pollPayout.PaymentInitiationID,
			getPIStatusFromPayment(payment.Status),
		); err != nil {
			return "", err
		}

		paymentID = payment.ID.String()
		isFinal = isPaymentStatusFinal(payment.Status)

	case pollPayoutStatusResponse.Error != nil:
		// Means that the payment initiation failed, and we need to register
		// the error in the task as well as stopping the schedule polling.
		piErr = fmt.Errorf("%s", *pollPayoutStatusResponse.Error)
		isFinal = true
	}

	// Only delete the schedule if the status is final (not PENDING/PROCESSING)
	if !isFinal {
		// Intermediate status - continue polling
		return "", nil
	}

	// Final status - delete the related schedule
	if err := activities.TemporalScheduleDelete(
		infiniteRetryContext(ctx),
		pollPayout.ScheduleID,
	); err != nil {
		return "", err
	}

	if err := activities.StorageSchedulesDelete(
		infiniteRetryContext(ctx),
		pollPayout.ScheduleID,
	); err != nil {
		return "", err
	}

	return paymentID, piErr
}

const RunPollPayout = "PollPayout"
