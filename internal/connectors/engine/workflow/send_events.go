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
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:    &sendEvents.Account.ConnectorID,
				IdempotencyKey: sendEvents.Account.IdempotencyKey(),
				At:             workflow.Now(ctx).UTC(),
				Account:        sendEvents.Account,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.Balance != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:    &sendEvents.Balance.AccountID.ConnectorID,
				IdempotencyKey: sendEvents.Balance.IdempotencyKey(),
				At:             workflow.Now(ctx).UTC(),
				Balance:        sendEvents.Balance,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.BankAccount != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				IdempotencyKey: sendEvents.BankAccount.IdempotencyKey(),
				At:             workflow.Now(ctx).UTC(),
				BankAccount:    sendEvents.BankAccount,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.Payment != nil {
		for _, adjustment := range sendEvents.Payment.Adjustments {
			adj := adjustment
			err := activities.SendEvents(
				infiniteRetryContext(ctx),
				activities.SendEventsRequest{
					ConnectorID:    &sendEvents.Payment.ConnectorID,
					IdempotencyKey: adj.IdempotencyKey(),
					At:             workflow.Now(ctx).UTC(),
					Payment: &activities.SendEventsPayment{
						Payment:    *sendEvents.Payment,
						Adjustment: adj,
					},
				},
			)
			if err != nil {
				return err
			}
		}
	}

	if sendEvents.PaymentDeleted != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:    &sendEvents.PaymentDeleted.ConnectorID,
				IdempotencyKey: fmt.Sprintf("delete:%s", sendEvents.PaymentDeleted.String()),
				At:             workflow.Now(ctx).UTC(),
				PaymentDeleted: sendEvents.PaymentDeleted,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.ConnectorReset != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:    sendEvents.ConnectorReset,
				IdempotencyKey: fmt.Sprintf("%s-%s", sendEvents.ConnectorReset.String(), workflow.Now(ctx).UTC().Format(time.RFC3339Nano)),
				At:             workflow.Now(ctx).UTC(),
				ConnectorReset: sendEvents.ConnectorReset,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.PoolsCreation != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:    nil,
				IdempotencyKey: sendEvents.PoolsCreation.IdempotencyKey(),
				At:             workflow.Now(ctx).UTC(),
				PoolsCreation:  sendEvents.PoolsCreation,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.PoolsDeletion != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:    nil,
				IdempotencyKey: sendEvents.PoolsDeletion.String(),
				At:             workflow.Now(ctx).UTC(),
				PoolsDeletion:  sendEvents.PoolsDeletion,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.PaymentInitiation != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:       &sendEvents.PaymentInitiation.ConnectorID,
				IdempotencyKey:    sendEvents.PaymentInitiation.IdempotencyKey(),
				At:                workflow.Now(ctx).UTC(),
				PaymentInitiation: sendEvents.PaymentInitiation,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.PaymentInitiationAdjustment != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:                 &sendEvents.PaymentInitiationAdjustment.ID.PaymentInitiationID.ConnectorID,
				IdempotencyKey:              sendEvents.PaymentInitiationAdjustment.IdempotencyKey(),
				At:                          workflow.Now(ctx).UTC(),
				PaymentInitiationAdjustment: sendEvents.PaymentInitiationAdjustment,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.PaymentInitiationRelatedPayment != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:                     &sendEvents.PaymentInitiationRelatedPayment.PaymentInitiationID.ConnectorID,
				IdempotencyKey:                  sendEvents.PaymentInitiationRelatedPayment.IdempotencyKey(),
				At:                              workflow.Now(ctx).UTC(),
				PaymentInitiationRelatedPayment: sendEvents.PaymentInitiationRelatedPayment,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.Task != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:    sendEvents.Task.ConnectorID,
				IdempotencyKey: sendEvents.Task.IdempotencyKey(),
				At:             workflow.Now(ctx).UTC(),
				Task:           sendEvents.Task,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.UserPendingDisconnect != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:           &sendEvents.UserPendingDisconnect.ConnectorID,
				IdempotencyKey:        sendEvents.UserPendingDisconnect.IdempotencyKey(),
				At:                    workflow.Now(ctx).UTC(),
				UserPendingDisconnect: sendEvents.UserPendingDisconnect,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.UserConnectionDisconnected != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:                &sendEvents.UserConnectionDisconnected.ConnectorID,
				IdempotencyKey:             sendEvents.UserConnectionDisconnected.IdempotencyKey(),
				At:                         workflow.Now(ctx).UTC(),
				UserConnectionDisconnected: sendEvents.UserConnectionDisconnected,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.UserConnectionReconnected != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:               &sendEvents.UserConnectionReconnected.ConnectorID,
				IdempotencyKey:            sendEvents.UserConnectionReconnected.IdempotencyKey(),
				At:                        workflow.Now(ctx).UTC(),
				UserConnectionReconnected: sendEvents.UserConnectionReconnected,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.UserDisconnected != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:      &sendEvents.UserDisconnected.ConnectorID,
				IdempotencyKey:   sendEvents.UserDisconnected.IdempotencyKey(),
				At:               workflow.Now(ctx).UTC(),
				UserDisconnected: sendEvents.UserDisconnected,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.UserLinkStatus != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:    &sendEvents.UserLinkStatus.ConnectorID,
				IdempotencyKey: sendEvents.UserLinkStatus.IdempotencyKey(),
				At:             workflow.Now(ctx).UTC(),
				UserLinkStatus: sendEvents.UserLinkStatus,
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.UserConnectionDataSynced != nil {
		err := activities.SendEvents(
			infiniteRetryContext(ctx),
			activities.SendEventsRequest{
				ConnectorID:              &sendEvents.UserConnectionDataSynced.ConnectorID,
				IdempotencyKey:           sendEvents.UserConnectionDataSynced.IdempotencyKey(),
				At:                       workflow.Now(ctx).UTC(),
				UserConnectionDataSynced: sendEvents.UserConnectionDataSynced,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

const RunSendEvents = "RunSendEvents"
