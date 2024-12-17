package workflow

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
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
			payment := models.FromPSPPaymentToPayment(*createPayoutResponse.Payment, createPayout.ConnectorID)

			if err := w.storePIPaymentWithStatus(
				ctx,
				payment,
				createPayout.PaymentInitiationID,
				getPIStatusFromPayment(payment.Status),
				createPayout.ConnectorID,
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
			config, err := w.plugins.GetConfig(createPayout.ConnectorID)
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
						Workflow: RunPollTransfer,
						Args: []interface{}{
							PollTransfer{
								TaskID:              createPayout.TaskID,
								ConnectorID:         createPayout.ConnectorID,
								PaymentInitiationID: createPayout.PaymentInitiationID,
								TransferID:          *createPayoutResponse.PollingPayoutID,
								ScheduleID:          scheduleID,
							},
						},
						TaskQueue: w.getConnectorTaskQueue(createPayout.ConnectorID),
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
		err = w.addPIAdjustment(
			ctx,
			models.PaymentInitiationAdjustmentID{
				PaymentInitiationID: createPayout.PaymentInitiationID,
				CreatedAt:           workflow.Now(ctx),
				Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
			},
			pi.Amount,
			&pi.Asset,
			err,
			nil,
		)
		if err != nil {
			return err
		}

		return errPlugin
	}
}

const RunCreatePayout = "CreatePayout"
