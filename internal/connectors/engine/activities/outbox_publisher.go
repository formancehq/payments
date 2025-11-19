package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
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

	// Collect successful events for batch processing
	var successfulEventIDs []models.EventID
	var successfulEventsSent []models.EventSent

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

		// Success - collect for batch deletion and recording
		eventSent := models.EventSent{
			ID:          event.ID,
			ConnectorID: event.ConnectorID,
			SentAt:      time.Now().UTC(),
		}
		successfulEventIDs = append(successfulEventIDs, event.ID)
		successfulEventsSent = append(successfulEventsSent, eventSent)
	}

	// Batch delete and record successful events
	if len(successfulEventIDs) > 0 {
		if err := a.storage.OutboxEventsDeleteAndRecordSent(ctx, successfulEventIDs, successfulEventsSent); err != nil {
			return fmt.Errorf("failed to delete outbox events and record sent: %w", err)
		}
	}

	return nil
}

func (a Activities) processOutboxEvent(ctx context.Context, event models.OutboxEvent) error {
	// Create the event message
	eventMessage := publish.EventMessage{
		IdempotencyKey: event.ID.EventIdempotencyKey,
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           event.EventType,
		Payload:        event.Payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

var OutboxPublishPendingEventsActivity = Activities{}.OutboxPublishPendingEvents

func OutboxPublishPendingEvents(ctx workflow.Context, limit int) error {
	return executeActivity(ctx, OutboxPublishPendingEventsActivity, nil, limit)
}
