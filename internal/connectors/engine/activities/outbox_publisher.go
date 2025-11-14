package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) OutboxPublishPendingEvents(ctx context.Context, limit int) error {
	// Poll pending events from outbox
	outboxEvents, err := a.storage.OutboxEventsPollPending(ctx, limit)
	if err != nil {
		return fmt.Errorf("failed to poll pending outbox events: %w", err)
	}

	if len(outboxEvents) == 0 {
		return nil // No events to process
	}

	// Process each event
	for _, event := range outboxEvents {
		if err := a.processOutboxEvent(ctx, event); err != nil {
			// Increment retry count (OutboxEventsMarkFailed handles status based on retry count)
			retryCount := event.RetryCount + 1

			if markErr := a.storage.OutboxEventsMarkFailed(ctx, event.ID, retryCount, err); markErr != nil {
				return fmt.Errorf("failed to update event retry count: %w", markErr)
			}
			continue
		}

		// Success - delete from outbox and record in events_sent in same transaction
		if err := a.deleteOutboxAndRecordSent(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

func (a Activities) processOutboxEvent(ctx context.Context, event models.OutboxEvent) error {
	eventType, err := mapOutboxEventTypeToEventType(event.EventType)
	if err != nil {
		return err
	}

	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return &OutboxInvalidPayloadError{
			EventType: event.EventType,
			OutboxID:  event.ID,
			Err:       err,
		}
	}

	// Create the event message
	eventMessage := publish.EventMessage{
		IdempotencyKey: event.IdempotencyKey,
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           eventType,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) deleteOutboxAndRecordSent(ctx context.Context, event models.OutboxEvent) error {
	// Record in events_sent table and delete from outbox in same transaction
	idempotencyKey := event.IdempotencyKey
	eventSent := models.EventSent{
		ID: models.EventID{
			EventIdempotencyKey: idempotencyKey,
			ConnectorID:         event.ConnectorID,
		},
		ConnectorID: event.ConnectorID,
		SentAt:      time.Now().UTC(),
	}

	// Delete from outbox and record in events_sent atomically
	if err := a.storage.OutboxEventsDeleteAndRecordSent(ctx, event.ID, eventSent); err != nil {
		return fmt.Errorf("failed to delete outbox event and record sent: %w", err)
	}

	return nil
}

var OutboxPublishPendingEventsActivity = Activities{}.OutboxPublishPendingEvents

func OutboxPublishPendingEvents(ctx workflow.Context, limit int) error {
	return executeActivity(ctx, OutboxPublishPendingEventsActivity, nil, limit)
}

func mapOutboxEventTypeToEventType(outboxEventType string) (string, error) {
	var eventType string
	switch outboxEventType {
	case models.OUTBOX_EVENT_ACCOUNT_SAVED:
		eventType = events.EventTypeSavedAccounts
	case models.OUTBOX_EVENT_BALANCE_SAVED:
		eventType = events.EventTypeSavedBalances
	case models.OUTBOX_EVENT_PAYMENT_SAVED:
		eventType = events.EventTypeSavedPayments
	case models.OUTBOX_EVENT_PAYMENT_DELETED:
		eventType = events.EventTypeDeletedPayments
	case models.OUTBOX_EVENT_BANK_ACCOUNT_SAVED:
		eventType = events.EventTypeSavedBankAccount
	case models.OUTBOX_EVENT_TASK_UPDATED:
		eventType = events.EventTypeUpdatedTask
	case models.OUTBOX_EVENT_CONNECTOR_RESET:
		eventType = events.EventTypeConnectorReset
	case models.OUTBOX_EVENT_POOL_SAVED:
		eventType = events.EventTypeSavedPool
	case models.OUTBOX_EVENT_POOL_DELETED:
		eventType = events.EventTypeDeletePool
	case models.OUTBOX_EVENT_PAYMENT_INITIATION_SAVED:
		eventType = events.EventTypeSavedPaymentInitiation
	case models.OUTBOX_EVENT_PAYMENT_INITIATION_ADJUSTMENT_SAVED:
		eventType = events.EventTypeSavedPaymentInitiationAdjustment
	case models.OUTBOX_EVENT_PAYMENT_INITIATION_RELATED_PAYMENT_SAVED:
		eventType = events.EventTypeSavedPaymentInitiationRelatedPayment
	case models.OUTBOX_EVENT_USER_LINK_STATUS:
		eventType = events.EventTypeOpenBankingUserLinkStatus
	case models.OUTBOX_EVENT_USER_CONNECTION_DATA_SYNCED:
		eventType = events.EventTypeOpenBankingUserConnectionDataSynced
	case models.OUTBOX_EVENT_USER_CONNECTION_PENDING_DISCONNECT:
		eventType = events.EventTypeOpenBankingUserConnectionPendingDisconnect
	case models.OUTBOX_EVENT_USER_CONNECTION_DISCONNECTED:
		eventType = events.EventTypeOpenBankingUserConnectionDisconnected
	case models.OUTBOX_EVENT_USER_CONNECTION_RECONNECTED:
		eventType = events.EventTypeOpenBankingUserConnectionReconnected
	case models.OUTBOX_EVENT_USER_DISCONNECTED:
		eventType = events.EventTypeOpenBankingUserDisconnected
	default:
		return "", &OutboxEventUnknownEventTypeError{outboxEventType}
	}
	return eventType, nil
}

type OutboxEventUnknownEventTypeError struct {
	Type string
}

func (e *OutboxEventUnknownEventTypeError) Error() string {
	return "unknown outbox event type, type=" + e.Type
}

func (e *OutboxEventUnknownEventTypeError) NonRetryable() {}

type OutboxEventInvalidIdempotencyKeyError struct {
	EventType      string
	IdempotencyKey string
}

func (e *OutboxEventInvalidIdempotencyKeyError) Error() string {
	return fmt.Sprintf(
		"invalid idempotency key, key=%s, eventType=%s",
		e.IdempotencyKey,
		e.EventType,
	)
}

func (e *OutboxEventInvalidIdempotencyKeyError) NonRetryable() {}

type OutboxInvalidPayloadError struct {
	EventType string
	OutboxID  uuid.UUID
	Err       error
}

func (e *OutboxInvalidPayloadError) Error() string {
	return fmt.Sprintf(
		"invalid payload, error=%s, type=%s, id=%s",
		e.Err.Error(),
		e.EventType,
		e.OutboxID,
	)
}
func (e *OutboxInvalidPayloadError) NonRetryable() {}
