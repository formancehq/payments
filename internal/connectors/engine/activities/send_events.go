package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
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
	// Safely get logger - it may be nil in unit tests
	var logger interface{ Info(string, ...interface{}); Error(string, ...interface{}) }
	// Try to get activity logger, but don't panic if not in activity context
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Not an activity context, logger stays nil
				logger = nil
			}
		}()
		logger = activity.GetLogger(ctx)
	}()
	
	if logger != nil {
		logger.Info("SendEvents activity started",
			"idempotency_key", req.IdempotencyKey,
			"has_payment", req.Payment != nil)
	}

	eventID := models.EventID{
		EventIdempotencyKey: req.IdempotencyKey,
		ConnectorID:         req.ConnectorID,
	}
	isExisting, err := a.storage.EventsSentExists(ctx, eventID)
	if err != nil {
		if logger != nil {
			logger.Error("Failed to check if event exists", "error", err)
		}
		return temporalStorageError(err)
	}

	if isExisting {
		if logger != nil {
			logger.Info("Event already sent, skipping", "idempotency_key", req.IdempotencyKey)
		}
		// event was already sent; nothing to do
		return nil
	}

	if logger != nil {
		logger.Info("Sending event", "idempotency_key", req.IdempotencyKey)
	}
	// event was not sent yet
	if err := a.sendEvents(ctx, req, logger); err != nil {
		if logger != nil {
			logger.Error("Failed to send event", "error", err)
		}
		return err
	}

	if logger != nil {
		logger.Info("Storing event sent record", "idempotency_key", req.IdempotencyKey)
	}
	if err := a.storage.EventsSentUpsert(ctx, models.EventSent{
		ID:          eventID,
		ConnectorID: req.ConnectorID,
		SentAt:      req.At,
	}); err != nil {
		if logger != nil {
			logger.Error("Failed to store event sent record", "error", err)
		}
		return temporalStorageError(err)
	}

	if logger != nil {
		logger.Info("SendEvents activity completed successfully", "idempotency_key", req.IdempotencyKey)
	}
	return nil
}

func (a Activities) sendEvents(ctx context.Context, req SendEventsRequest, logger interface{ Info(string, ...interface{}); Error(string, ...interface{}) }) error {
	if req.Trade != nil {
		if logger != nil {
			logger.Info("Publishing trade event")
		}
		return a.events.Publish(ctx, a.events.NewEventSavedTrades(*req.Trade))
	}

	if req.Account != nil {
		if logger != nil {
			logger.Info("Publishing account event")
		}
		return a.events.Publish(ctx, a.events.NewEventSavedAccounts(*req.Account))
	}

	if req.Balance != nil {
		if logger != nil {
			logger.Info("Publishing balance event")
		}
		return a.events.Publish(ctx, a.events.NewEventSavedBalances(*req.Balance))
	}

	if req.BankAccount != nil {
		if logger != nil {
			logger.Info("Publishing bank account event")
		}
		ba, err := a.events.NewEventSavedBankAccounts(*req.BankAccount)
		if err != nil {
			return fmt.Errorf("failed to send bank account: %w", err)
		}
		return a.events.Publish(ctx, ba)
	}

	if req.Payment != nil {
		if logger != nil {
			logger.Info("Publishing payment event", 
				"payment_reference", req.Payment.Payment.Reference,
				"adjustment_reference", req.Payment.Adjustment.Reference)
		}
		err := a.events.Publish(ctx, a.events.NewEventSavedPayments(req.Payment.Payment, req.Payment.Adjustment))
		if err != nil {
			if logger != nil {
				logger.Error("Failed to publish payment event", "error", err)
			}
			return fmt.Errorf("failed to publish payment event: %w", err)
		}
		if logger != nil {
			logger.Info("Payment event published successfully")
		}
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
