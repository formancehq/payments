package activities

import (
	"context"
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
	// Create the event message
	eventMessage := publish.EventMessage{
		IdempotencyKey: event.IdempotencyKey,
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           event.EventType,
		Payload:        event.Payload,
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
