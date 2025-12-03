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

	Trade                           *models.Trade
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
	a.logger.Info("SendEvents activity started",
		"idempotency_key", req.IdempotencyKey,
		"has_payment", req.Payment != nil)

	eventID := models.EventID{
		EventIdempotencyKey: req.IdempotencyKey,
		ConnectorID:         req.ConnectorID,
	}
	isExisting, err := a.storage.EventsSentExists(ctx, eventID)
	if err != nil {
		a.logger.Error("Failed to check if event exists", "error", err)
		return temporalStorageError(err)
	}

	if isExisting {
		a.logger.Info("Event already sent, skipping", "idempotency_key", req.IdempotencyKey)
		// event was already sent; nothing to do
		return nil
	}

	a.logger.Info("Sending event", "idempotency_key", req.IdempotencyKey)
	// event was not sent yet
	if err := a.sendEvents(ctx, req); err != nil {
		a.logger.Error("Failed to send event", "error", err)
		return err
	}

	a.logger.Info("Storing event sent record", "idempotency_key", req.IdempotencyKey)
	if err := a.storage.EventsSentUpsert(ctx, models.EventSent{
		ID:          eventID,
		ConnectorID: req.ConnectorID,
		SentAt:      req.At,
	}); err != nil {
		a.logger.Error("Failed to store event sent record", "error", err)
		return temporalStorageError(err)
	}

	a.logger.Info("SendEvents activity completed successfully", "idempotency_key", req.IdempotencyKey)
	return nil
}

func (a Activities) sendEvents(ctx context.Context, req SendEventsRequest) error {
	if req.Trade != nil {
		a.logger.Info("Publishing trade event")
		return a.events.Publish(ctx, a.events.NewEventSavedTrades(*req.Trade))
	}

	if req.Account != nil {
		a.logger.Info("Publishing account event")
		return a.events.Publish(ctx, a.events.NewEventSavedAccounts(*req.Account))
	}

	if req.Balance != nil {
		a.logger.Info("Publishing balance event")
		return a.events.Publish(ctx, a.events.NewEventSavedBalances(*req.Balance))
	}

	if req.BankAccount != nil {
		a.logger.Info("Publishing bank account event")
		ba, err := a.events.NewEventSavedBankAccounts(*req.BankAccount)
		if err != nil {
			return fmt.Errorf("failed to send bank account: %w", err)
		}
		return a.events.Publish(ctx, ba)
	}

	if req.Payment != nil {
		a.logger.Info("Publishing payment event",
			"payment_reference", req.Payment.Payment.Reference,
			"adjustment_reference", req.Payment.Adjustment.Reference)
		err := a.events.Publish(ctx, a.events.NewEventSavedPayments(req.Payment.Payment, req.Payment.Adjustment))
		if err != nil {
			a.logger.Error("Failed to publish payment event", "error", err)
			return fmt.Errorf("failed to publish payment event: %w", err)
		}
		a.logger.Info("Payment event published successfully")
		return nil
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
