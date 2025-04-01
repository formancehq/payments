package workflow

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type sendEventActivityFunction func(ctx workflow.Context) error

type SendEventPaymentInitiationAdjustment struct {
	PaymentInitiation           *models.PaymentInitiation
	PaymentInitiationAdjustment *models.PaymentInitiationAdjustment
}

type SendEventPaymentInitiationRelatedPayment struct {
	PaymentInitiation               *models.PaymentInitiation
	PaymentInitiationRelatedPayment *models.PaymentInitiationRelatedPayments
	Status                          models.PaymentInitiationAdjustmentStatus
}

type SendEvents struct {
	Account                                  *models.Account
	Balance                                  *models.Balance
	BankAccount                              *models.BankAccount
	Payment                                  *models.Payment
	ConnectorReset                           *models.ConnectorID
	PoolsCreation                            *models.Pool
	PoolsDeletion                            *uuid.UUID
	PaymentInitiation                        *models.PaymentInitiation
	SendEventPaymentInitiationAdjustment     *SendEventPaymentInitiationAdjustment
	SendEventPaymentInitiationRelatedPayment *SendEventPaymentInitiationRelatedPayment
}

func (w Workflow) runSendEvents(
	ctx workflow.Context,
	sendEvents SendEvents,
) error {
	if sendEvents.Account != nil {
		err := sendEvent(
			ctx,
			sendEvents.Account.IdempotencyKey(),
			&sendEvents.Account.ConnectorID,
			func(ctx workflow.Context) error {
				return activities.EventsSendAccount(
					infiniteRetryContext(ctx),
					*sendEvents.Account,
				)
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.Balance != nil {
		err := sendEvent(
			ctx,
			sendEvents.Balance.IdempotencyKey(),
			&sendEvents.Balance.AccountID.ConnectorID,
			func(ctx workflow.Context) error {
				return activities.EventsSendBalance(
					infiniteRetryContext(ctx),
					*sendEvents.Balance,
				)
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.BankAccount != nil {
		err := sendEvent(
			ctx,
			sendEvents.BankAccount.IdempotencyKey(),
			nil,
			func(ctx workflow.Context) error {
				return activities.EventsSendBankAccount(
					infiniteRetryContext(ctx),
					*sendEvents.BankAccount,
				)
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.Payment != nil {
		for _, adjustment := range sendEvents.Payment.Adjustments {
			err := sendEvent(
				ctx,
				adjustment.IdempotencyKey(),
				&sendEvents.Payment.ConnectorID,
				func(ctx workflow.Context) error {
					return activities.EventsSendPayment(
						infiniteRetryContext(ctx),
						*sendEvents.Payment,
						adjustment,
					)
				},
			)
			if err != nil {
				return err
			}
		}
	}

	if sendEvents.ConnectorReset != nil {
		now := workflow.Now(ctx).UTC()
		err := sendEvent(
			ctx,
			fmt.Sprintf("%s-%s", sendEvents.ConnectorReset.String(), now.Format(time.RFC3339Nano)),
			nil,
			func(ctx workflow.Context) error {
				return activities.EventsSendConnectorReset(
					infiniteRetryContext(ctx),
					*sendEvents.ConnectorReset,
					now,
				)
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.PoolsCreation != nil {
		err := sendEvent(
			ctx,
			sendEvents.PoolsCreation.IdempotencyKey(),
			nil,
			func(ctx workflow.Context) error {
				return activities.EventsSendPoolCreation(
					infiniteRetryContext(ctx),
					*sendEvents.PoolsCreation,
				)
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.PoolsDeletion != nil {
		err := sendEvent(
			ctx,
			sendEvents.PoolsDeletion.String(),
			nil,
			func(ctx workflow.Context) error {
				return activities.EventsSendPoolDeletion(
					infiniteRetryContext(ctx),
					*sendEvents.PoolsDeletion,
					workflow.Now(ctx).UTC(),
				)
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.PaymentInitiation != nil {
		err := sendEvent(
			ctx,
			sendEvents.PaymentInitiation.IdempotencyKey(),
			&sendEvents.PaymentInitiation.ConnectorID,
			func(ctx workflow.Context) error {
				return activities.EventsSendPaymentInitiation(
					infiniteRetryContext(ctx),
					*sendEvents.PaymentInitiation,
				)
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.SendEventPaymentInitiationAdjustment != nil &&
		sendEvents.SendEventPaymentInitiationAdjustment.PaymentInitiation != nil &&
		sendEvents.SendEventPaymentInitiationAdjustment.PaymentInitiationAdjustment != nil {
		err := sendEvent(
			ctx,
			sendEvents.SendEventPaymentInitiationAdjustment.PaymentInitiationAdjustment.IdempotencyKey(),
			&sendEvents.SendEventPaymentInitiationAdjustment.PaymentInitiation.ConnectorID,
			func(ctx workflow.Context) error {
				return activities.EventsSendPaymentInitiationAdjustment(
					infiniteRetryContext(ctx),
					*sendEvents.SendEventPaymentInitiationAdjustment.PaymentInitiationAdjustment,
					*sendEvents.SendEventPaymentInitiationAdjustment.PaymentInitiation,
				)
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.SendEventPaymentInitiationRelatedPayment != nil &&
		sendEvents.SendEventPaymentInitiationRelatedPayment.PaymentInitiation != nil &&
		sendEvents.SendEventPaymentInitiationRelatedPayment.PaymentInitiationRelatedPayment != nil {
		err := sendEvent(
			ctx,
			sendEvents.SendEventPaymentInitiationRelatedPayment.PaymentInitiationRelatedPayment.IdempotencyKey(),
			&sendEvents.SendEventPaymentInitiationRelatedPayment.PaymentInitiation.ConnectorID,
			func(ctx workflow.Context) error {
				return activities.EventsSendPaymentInitiationRelatedPayment(
					infiniteRetryContext(ctx),
					*sendEvents.SendEventPaymentInitiationRelatedPayment.PaymentInitiationRelatedPayment,
					*sendEvents.SendEventPaymentInitiationRelatedPayment.PaymentInitiation,
					sendEvents.SendEventPaymentInitiationRelatedPayment.Status,
				)
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

const RunSendEvents = "RunSendEvents"

func sendEvent(
	ctx workflow.Context,
	idempotencyKey string,
	connectorID *models.ConnectorID,
	fn sendEventActivityFunction,
) error {
	isExisting, err := activities.StorageEventsSentExists(
		infiniteRetryContext(ctx),
		idempotencyKey,
		connectorID,
	)
	if err != nil {
		return err
	}

	if !isExisting {
		// event was not sent yet
		if err := fn(ctx); err != nil {
			return err
		}

		if err := activities.StorageEventsSentStore(
			infiniteRetryContext(ctx),
			models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: idempotencyKey,
					ConnectorID:         connectorID,
				},
				ConnectorID: connectorID,
				SentAt:      workflow.Now(ctx).UTC(),
			},
		); err != nil {
			return err
		}
	}

	return nil
}
