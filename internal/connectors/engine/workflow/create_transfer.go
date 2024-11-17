package workflow

import (
	"fmt"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type CreateTransfer struct {
	TaskID              models.TaskID
	ConnectorID         models.ConnectorID
	PaymentInitiationID models.PaymentInitiationID
}

func (w Workflow) runCreateTransfer(
	ctx workflow.Context,
	createTransfer CreateTransfer,
) error {
	err := w.createTransfer(ctx, createTransfer)
	if err != nil {
		errUpdateTask := w.updateTasksError(
			ctx,
			createTransfer.TaskID,
			createTransfer.ConnectorID,
			err,
		)
		if errUpdateTask != nil {
			return errUpdateTask
		}

		return err
	}

	return nil
}

func (w Workflow) createTransfer(
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

	pspPI, err := w.getPSPPI(ctx, pi)
	if err != nil {
		return err
	}

	err = w.addPIAdjustment(
		ctx,
		models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: createTransfer.PaymentInitiationID,
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

	createTransferResponse, errPlugin := activities.PluginCreateTransfer(
		infiniteRetryContext(ctx),
		createTransfer.ConnectorID,
		models.CreateTransferRequest{
			PaymentInitiation: pspPI,
		},
	)
	switch errPlugin {
	case nil:
		if createTransferResponse.Payment != nil {
			// payment is already available, storing it
			payment := models.FromPSPPaymentToPayment(*createTransferResponse.Payment, createTransfer.ConnectorID)

			if err := w.storePIPaymentWithStatus(
				ctx,
				payment,
				createTransfer.PaymentInitiationID,
				getPIStatusFromPayment(payment.Status),
				createTransfer.ConnectorID,
			); err != nil {
				return err
			}

			return w.updateTaskSuccess(
				ctx,
				createTransfer.TaskID,
				createTransfer.ConnectorID,
				payment.ID.String(),
			)
		}

		if createTransferResponse.PollingTransferID != nil {
			// payment not yet available, waiting for the next polling
			config, err := w.plugins.GetConfig(createTransfer.ConnectorID)
			if err != nil {
				return err
			}

			scheduleID := fmt.Sprintf("polling-transfer-%s-%s-%s", w.stack, createTransfer.ConnectorID.String(), *createTransferResponse.PollingTransferID)
			scheduleID, err = activities.TemporalScheduleCreate(
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
								TaskID:              createTransfer.TaskID,
								ConnectorID:         createTransfer.ConnectorID,
								PaymentInitiationID: createTransfer.PaymentInitiationID,
								TransferID:          *createTransferResponse.PollingTransferID,
								ScheduleID:          scheduleID,
							},
						},
						TaskQueue: createTransfer.ConnectorID.String(),
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

			err = activities.StorageSchedulesStore(
				infiniteRetryContext(ctx),
				models.Schedule{
					ID:          scheduleID,
					ConnectorID: createTransfer.ConnectorID,
					CreatedAt:   workflow.Now(ctx).UTC(),
				})
			if err != nil {
				return err
			}
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
			pi.Amount,
			&pi.Asset,
			errPlugin,
			nil,
		)
		if err != nil {
			return err
		}

		return errPlugin
	}
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
	}

	pspPI := models.FromPaymentInitiationToPSPPaymentInitiation(pi, models.ToPSPAccount(sourceAccount), models.ToPSPAccount(destinationAccount))

	return pspPI, nil
}

const RunCreateTransfer = "CreateTransfer"

func (w Workflow) addPIAdjustment(
	ctx workflow.Context,
	adjustmentID models.PaymentInitiationAdjustmentID,
	amount *big.Int,
	asset *string,
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
