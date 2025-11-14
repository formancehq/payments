package workflow

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type SendEvents struct {
	Account                         *models.Account
	Balance                         *models.Balance
	BankAccount                     *models.BankAccount
	Payment                         *models.Payment
	PaymentDeleted                  *models.PaymentID
	Trade                           *models.Trade
	ConnectorReset                  *models.ConnectorID
	PoolsCreation                   *models.Pool
	PoolsDeletion                   *uuid.UUID
	PaymentInitiation               *models.PaymentInitiation
	PaymentInitiationAdjustment     *models.PaymentInitiationAdjustment
	PaymentInitiationRelatedPayment *models.PaymentInitiationRelatedPayments
	UserPendingDisconnect           *models.UserConnectionPendingDisconnect
	UserDisconnected                *models.UserDisconnected
	UserConnectionDisconnected      *models.UserConnectionDisconnected
	UserConnectionReconnected       *models.UserConnectionReconnected
	UserLinkStatus                  *models.UserLinkSessionFinished
	UserConnectionDataSynced        *models.UserConnectionDataSynced
	Task                            *models.Task
}

func (w Workflow) runSendEvents(
	ctx workflow.Context,
	sendEvents SendEvents,
) error {
	if sendEvents.Account != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:    &sendEvents.Account.ConnectorID,
			IdempotencyKey: sendEvents.Account.IdempotencyKey(),
			Account:        sendEvents.Account,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.Balance != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:    &sendEvents.Balance.AccountID.ConnectorID,
			IdempotencyKey: sendEvents.Balance.IdempotencyKey(),
			Balance:        sendEvents.Balance,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.BankAccount != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			IdempotencyKey: sendEvents.BankAccount.IdempotencyKey(),
			BankAccount:    sendEvents.BankAccount,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.Payment != nil {
		// Log adjustment count for debugging
		workflow.GetLogger(ctx).Info("Processing payment event",
			"reference", sendEvents.Payment.Reference,
			"adjustments_count", len(sendEvents.Payment.Adjustments))

		if len(sendEvents.Payment.Adjustments) == 0 {
			workflow.GetLogger(ctx).Warn("Payment has no adjustments, no events will be sent",
				"payment_id", sendEvents.Payment.ID.String(),
				"payment_reference", sendEvents.Payment.Reference)
		}

		for i, adjustment := range sendEvents.Payment.Adjustments {
			workflow.GetLogger(ctx).Info("Calling sendEvent for adjustment",
				"adjustment_index", i,
				"adjustment_reference", adjustment.Reference,
				"adjustment_idempotency", adjustment.IdempotencyKey())

			err := sendEvent(ctx, activities.SendEventsRequest{
				ConnectorID:    &sendEvents.Payment.ConnectorID,
				IdempotencyKey: adjustment.IdempotencyKey(),
				Payment: &activities.SendEventsPayment{
					Payment:    *sendEvents.Payment,
					Adjustment: adjustment,
				},
			})
			if err != nil {
				workflow.GetLogger(ctx).Error("sendEvent failed", "error", err)
				return err
			}
			workflow.GetLogger(ctx).Info("sendEvent completed successfully", "adjustment_index", i)
		}
	}

	if sendEvents.Trade != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:    &sendEvents.Trade.ConnectorID,
			IdempotencyKey: sendEvents.Trade.IdempotencyKey(),
			Trade:          sendEvents.Trade,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.PaymentDeleted != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:    &sendEvents.PaymentDeleted.ConnectorID,
			IdempotencyKey: fmt.Sprintf("delete:%s", sendEvents.PaymentDeleted.String()),
			PaymentDeleted: sendEvents.PaymentDeleted,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.ConnectorReset != nil {
		now := workflow.Now(ctx).UTC()
		err := sendEvent(ctx, activities.SendEventsRequest{
			IdempotencyKey: fmt.Sprintf("%s-%s", sendEvents.ConnectorReset.String(), now.Format(time.RFC3339Nano)),
			ConnectorReset: sendEvents.ConnectorReset,
			At:             now,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.PoolsCreation != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			IdempotencyKey: sendEvents.PoolsCreation.IdempotencyKey(),
			PoolsCreation:  sendEvents.PoolsCreation,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.PoolsDeletion != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			IdempotencyKey: sendEvents.PoolsDeletion.String(),
			PoolsDeletion:  sendEvents.PoolsDeletion,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.PaymentInitiation != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:       &sendEvents.PaymentInitiation.ConnectorID,
			IdempotencyKey:    sendEvents.PaymentInitiation.IdempotencyKey(),
			PaymentInitiation: sendEvents.PaymentInitiation,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.PaymentInitiationAdjustment != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:                 &sendEvents.PaymentInitiationAdjustment.ID.PaymentInitiationID.ConnectorID,
			IdempotencyKey:              sendEvents.PaymentInitiationAdjustment.IdempotencyKey(),
			PaymentInitiationAdjustment: sendEvents.PaymentInitiationAdjustment,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.PaymentInitiationRelatedPayment != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:                     &sendEvents.PaymentInitiationRelatedPayment.PaymentInitiationID.ConnectorID,
			IdempotencyKey:                  sendEvents.PaymentInitiationRelatedPayment.IdempotencyKey(),
			PaymentInitiationRelatedPayment: sendEvents.PaymentInitiationRelatedPayment,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.Task != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:    sendEvents.Task.ConnectorID,
			IdempotencyKey: sendEvents.Task.IdempotencyKey(),
			Task:           sendEvents.Task,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.UserPendingDisconnect != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:           &sendEvents.UserPendingDisconnect.ConnectorID,
			IdempotencyKey:        sendEvents.UserPendingDisconnect.IdempotencyKey(),
			UserPendingDisconnect: sendEvents.UserPendingDisconnect,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.UserConnectionDisconnected != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:                &sendEvents.UserConnectionDisconnected.ConnectorID,
			IdempotencyKey:             sendEvents.UserConnectionDisconnected.IdempotencyKey(),
			UserConnectionDisconnected: sendEvents.UserConnectionDisconnected,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.UserConnectionReconnected != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:               &sendEvents.UserConnectionReconnected.ConnectorID,
			IdempotencyKey:            sendEvents.UserConnectionReconnected.IdempotencyKey(),
			UserConnectionReconnected: sendEvents.UserConnectionReconnected,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.UserDisconnected != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:      &sendEvents.UserDisconnected.ConnectorID,
			IdempotencyKey:   sendEvents.UserDisconnected.IdempotencyKey(),
			UserDisconnected: sendEvents.UserDisconnected,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.UserLinkStatus != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:    &sendEvents.UserLinkStatus.ConnectorID,
			IdempotencyKey: sendEvents.UserLinkStatus.IdempotencyKey(),
			UserLinkStatus: sendEvents.UserLinkStatus,
		})
		if err != nil {
			return err
		}
	}

	if sendEvents.UserConnectionDataSynced != nil {
		err := sendEvent(ctx, activities.SendEventsRequest{
			ConnectorID:              &sendEvents.UserConnectionDataSynced.ConnectorID,
			IdempotencyKey:           sendEvents.UserConnectionDataSynced.IdempotencyKey(),
			UserConnectionDataSynced: sendEvents.UserConnectionDataSynced,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

const RunSendEvents = "RunSendEvents"

func sendEvent(
	ctx workflow.Context,
	req activities.SendEventsRequest,
) error {
	if req.At.IsZero() {
		req.At = workflow.Now(ctx).UTC()
	}

	return activities.SendEvents(
		infiniteRetryContext(ctx),
		req,
	)
}
