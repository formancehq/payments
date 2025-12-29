package workflow

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-faster/errors"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type sendEventActivityFunction func(ctx workflow.Context) error

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

	if sendEvents.PaymentInitiationAdjustment != nil {
		err := sendEvent(
			ctx,
			sendEvents.PaymentInitiationAdjustment.IdempotencyKey(),
			&sendEvents.PaymentInitiationAdjustment.ID.PaymentInitiationID.ConnectorID,
			func(ctx workflow.Context) error {
				return activities.EventsSendPaymentInitiationAdjustment(
					infiniteRetryContext(ctx),
					*sendEvents.PaymentInitiationAdjustment,
				)
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.PaymentInitiationRelatedPayment != nil {
		err := sendEvent(
			ctx,
			sendEvents.PaymentInitiationRelatedPayment.IdempotencyKey(),
			&sendEvents.PaymentInitiationRelatedPayment.PaymentInitiationID.ConnectorID,
			func(ctx workflow.Context) error {
				return activities.EventsSendPaymentInitiationRelatedPayment(
					infiniteRetryContext(ctx),
					*sendEvents.PaymentInitiationRelatedPayment,
				)
			},
		)
		if err != nil {
			return err
		}
	}

	if sendEvents.Task != nil {
		err := sendEvent(
			ctx,
			sendEvents.Task.IdempotencyKey(),
			sendEvents.Task.ConnectorID,
			func(ctx workflow.Context) error {
				return activities.EventsSendTaskUpdated(
					infiniteRetryContext(ctx),
					*sendEvents.Task,
				)
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

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

func (w Workflow) runSendEventAsChildWorkflow(ctx workflow.Context, wg workflow.WaitGroup, evts SendEvents, errChan chan error) chan error {
	wg.Add(1)
	workflow.Go(ctx, func(ctx workflow.Context) {
		defer wg.Done()

		if err := workflow.ExecuteChildWorkflow(
			workflow.WithChildOptions(
				ctx,
				workflow.ChildWorkflowOptions{
					TaskQueue:         w.getDefaultTaskQueue(),
					ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
					SearchAttributes: map[string]interface{}{
						SearchAttributeStack: w.stack,
					},
				},
			),
			RunSendEvents, // nolint: staticcheck // ignore deprecated
			evts,
		).Get(ctx, nil); err != nil {
			errChan <- errors.Wrap(err, "sending events")
		}
	})
	return errChan
}

// RunSendEvents
// Deprecated: should not be used after version 3.0; we keep it in 3.1 for ongoing workflows.
const RunSendEvents = "RunSendEvents" //nolint:staticcheck // ignore deprecation
