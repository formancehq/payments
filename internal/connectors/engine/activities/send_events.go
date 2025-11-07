package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type SendEventsPayment struct {
	Payment    models.Payment
	Adjustment models.PaymentAdjustment
}

type SendEventsRequest struct {
	ConnectorID    *models.ConnectorID
	IdempotencyKey string
	At             time.Time

	Account                         *models.Account
	Balance                         *models.Balance
	BankAccount                     *models.BankAccount
	Payment                         *SendEventsPayment
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

func (a Activities) SendEvents(ctx context.Context, req SendEventsRequest) error {
	eventID := models.EventID{
		EventIdempotencyKey: req.IdempotencyKey,
		ConnectorID:         req.ConnectorID,
	}
	isExisting, err := a.storage.EventsSentExists(ctx, eventID)
	if err != nil {
		return temporalStorageError(err)
	}

	if isExisting {
		// event was already sent; nothing to do
		return nil
	}
	
	// event was not sent yet
	if err := a.sendEvents(ctx, req); err != nil {
		return err
	}

	if err := a.storage.EventsSentUpsert(ctx, models.EventSent{
		ID:          eventID,
		ConnectorID: req.ConnectorID,
		SentAt:      req.At,
	}); err != nil {
		return temporalStorageError(err)
	}

	return nil
}

func (a Activities) sendEvents(ctx context.Context, req SendEventsRequest) error {
	if req.Account != nil {
		return a.events.Publish(ctx, a.events.NewEventSavedAccounts(*req.Account))
	}

	if req.Balance != nil {
		return a.events.Publish(ctx, a.events.NewEventSavedBalances(*req.Balance))
	}

	if req.BankAccount != nil {
		ba, err := a.events.NewEventSavedBankAccounts(*req.BankAccount)
		if err != nil {
			return fmt.Errorf("failed to send bank account: %w", err)
		}
		return a.events.Publish(ctx, ba)
	}

	if req.Payment != nil {
		return a.events.Publish(ctx, a.events.NewEventSavedPayments(req.Payment.Payment, req.Payment.Adjustment))
	}

	if req.PaymentDeleted != nil {
		return a.events.Publish(ctx, a.events.NewEventPaymentDeleted(*req.PaymentDeleted))
	}

	if req.ConnectorReset != nil {
		return a.events.Publish(ctx, a.events.NewEventResetConnector(*req.ConnectorReset, req.At))
	}

	if req.PoolsCreation != nil {
		return a.events.Publish(ctx, a.events.NewEventSavedPool(*req.PoolsCreation))
	}

	if req.PoolsDeletion != nil {
		return a.events.Publish(ctx, a.events.NewEventDeletePool(*req.PoolsDeletion))
	}

	if req.PaymentInitiation != nil {
		return a.events.Publish(ctx, a.events.NewEventSavedPaymentInitiation(*req.PaymentInitiation))
	}

	if req.PaymentInitiationAdjustment != nil {
		return a.events.Publish(ctx, a.events.NewEventSavedPaymentInitiationAdjustment(*req.PaymentInitiationAdjustment))
	}

	if req.PaymentInitiationRelatedPayment != nil {
		return a.events.Publish(ctx, a.events.NewEventSavedPaymentInitiationRelatedPayment(*req.PaymentInitiationRelatedPayment))
	}

	if req.UserPendingDisconnect != nil {
		return a.events.Publish(ctx, a.events.NewEventOpenBankingUserPendingDisconnect(*req.UserPendingDisconnect))
	}

	if req.UserDisconnected != nil {
		return a.events.Publish(ctx, a.events.NewEventOpenBankingUserDisconnected(*req.UserDisconnected))
	}

	if req.UserConnectionDisconnected != nil {
		return a.events.Publish(ctx, a.events.NewEventOpenBankingUserConnectionDisconnected(*req.UserConnectionDisconnected))
	}

	if req.UserConnectionReconnected != nil {
		return a.events.Publish(ctx, a.events.NewEventOpenBankingUserConnectionReconnected(*req.UserConnectionReconnected))
	}

	if req.UserLinkStatus != nil {
		return a.events.Publish(ctx, a.events.NewEventOpenBankingUserLinkStatus(*req.UserLinkStatus))
	}

	if req.UserConnectionDataSynced != nil {
		return a.events.Publish(ctx, a.events.NewEventOpenBankingUserConnectionDataSynced(*req.UserConnectionDataSynced))
	}

	if req.Task != nil {
		return a.events.Publish(ctx, a.events.NewEventUpdatedTask(*req.Task))
	}

	return nil
}

var SendEventsActivity = Activities{}.SendEvents

func SendEvents(ctx workflow.Context, req SendEventsRequest) error {
	return executeActivity(ctx, SendEventsActivity, nil, req)
}
