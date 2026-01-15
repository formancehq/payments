package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
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
	err := w.createPayout(ctx, createPayout)
	if err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			createPayout.TaskID,
			&createPayout.ConnectorID,
			err,
		)
		if errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return nil
}

func (w Workflow) createPayout(
	ctx workflow.Context,
	createPayout CreatePayout,
) error {
	// Get the payment initiation
	pi, err := activities.StoragePaymentInitiationsGet(
		infiniteRetryContext(ctx),
		createPayout.PaymentInitiationID,
	)
	if err != nil {
		return err
	}

	// If the payout is scheduled in the future, we need to add a schedule
	// for processing adjustment to the payment initiation, and then sleep until
	// the scheduled time.
	now := workflow.Now(ctx)
	if !pi.ScheduledAt.IsZero() && pi.ScheduledAt.After(now) {
		err = w.addPIAdjustment(
			ctx,
			models.PaymentInitiationAdjustmentID{
				PaymentInitiationID: createPayout.PaymentInitiationID,
				CreatedAt:           workflow.Now(ctx),
				Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_SCHEDULED_FOR_PROCESSING,
			},
			pi.Amount,
			&pi.Asset,
			nil,
			map[string]string{
				"scheduledAt": pi.ScheduledAt.String(),
			},
		)
		if err != nil {
			return err
		}

		err = workflow.Sleep(ctx, pi.ScheduledAt.Sub(now))
		if err != nil {
			return err
		}
	}

	pspPI, err := w.getPSPPI(ctx, pi)
	if err != nil {
		return err
	}

	err = w.addPIAdjustment(
		ctx,
		models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: createPayout.PaymentInitiationID,
			CreatedAt:           workflow.Now(ctx),
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING,
		},
		pi.Amount,
		&pi.Asset,
		nil,
		nil,
	)
	if err != nil {
		return err
	}

	createPayoutResponse, errPlugin := activities.PluginCreatePayout(
		infiniteRetryContext(ctx),
		createPayout.ConnectorID,
		models.CreatePayoutRequest{
			PaymentInitiation: pspPI,
		},
	)

	switch errPlugin {
	case nil:
		if createPayoutResponse.Payment != nil {
			payment, err := models.FromPSPPaymentToPayment(*createPayoutResponse.Payment, createPayout.ConnectorID)
			if err != nil {
				return temporal.NewNonRetryableApplicationError(
					"failed to translate psp payment",
					ErrValidation,
					err,
				)
			}

			if err := w.storePIPaymentWithStatus(
				ctx,
				payment,
				createPayout.PaymentInitiationID,
				getPIStatusFromPayment(payment.Status),
			); err != nil {
				return err
			}

			return w.updateTaskSuccess(
				ctx,
				createPayout.TaskID,
				&createPayout.ConnectorID,
				payment.ID.String(),
			)
		}

		if createPayoutResponse.PollingPayoutID != nil {
			config, err := w.connectors.GetConfig(createPayout.ConnectorID)
			if err != nil {
				return err
			}

			scheduleID := fmt.Sprintf("polling-payout-%s-%s-%s", w.stack, createPayout.ConnectorID.String(), *createPayoutResponse.PollingPayoutID)

			err = activities.StorageSchedulesStore(
				infiniteRetryContext(ctx),
				models.Schedule{
					ID:          scheduleID,
					ConnectorID: createPayout.ConnectorID,
					CreatedAt:   workflow.Now(ctx).UTC(),
				})
			if err != nil {
				return err
			}

			err = activities.TemporalScheduleCreate(
				infiniteRetryContext(ctx),
				activities.ScheduleCreateOptions{
					ScheduleID: scheduleID,
					Interval: client.ScheduleIntervalSpec{
						Every: config.PollingPeriod,
					},
					Action: client.ScheduleWorkflowAction{
						Workflow: RunPollPayout,
						Args: []interface{}{
							PollPayout{
								TaskID:              createPayout.TaskID,
								ConnectorID:         createPayout.ConnectorID,
								PaymentInitiationID: createPayout.PaymentInitiationID,
								PayoutID:            *createPayoutResponse.PollingPayoutID,
								ScheduleID:          scheduleID,
							},
						},
						TaskQueue: w.getDefaultTaskQueue(),
						TypedSearchAttributes: temporal.NewSearchAttributes(
							temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet(scheduleID),
							temporal.NewSearchAttributeKeyKeyword(SearchAttributeStack).ValueSet(w.stack),
						),
					},
					Overlap:            enums.SCHEDULE_OVERLAP_POLICY_SKIP,
					TriggerImmediately: true,
					SearchAttributes: map[string]any{
						SearchAttributeScheduleID: scheduleID,
						SearchAttributeStack:      w.stack,
					},
				},
			)
			if err != nil {
				return err
			}

		}

		return nil

	default:
		// Temporal errors do not have a Cause method, so we need to unwrap them
		// to get the underlying error and not store the whole stack trace inside
		// the database.
		cause := errorsutils.Cause(errPlugin)
		err = w.addPIAdjustment(
			ctx,
			models.PaymentInitiationAdjustmentID{
				PaymentInitiationID: createPayout.PaymentInitiationID,
				CreatedAt:           workflow.Now(ctx),
				Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
			},
			pi.Amount,
			&pi.Asset,
			cause,
			nil,
		)
		if err != nil {
			return err
		}

		return errPlugin
	}
}

const RunCreatePayout = "CreatePayout"
