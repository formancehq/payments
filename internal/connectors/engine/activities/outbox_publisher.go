package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) OutboxPublishPendingEvents(ctx context.Context, limit int) error {
	// Poll pending events from outbox
	events, err := a.storage.OutboxEventsPollPending(ctx, limit)
	if err != nil {
		return fmt.Errorf("failed to poll pending outbox events: %w", err)
	}

	if len(events) == 0 {
		return nil // No events to process
	}

	// Process each event
	for _, event := range events {
		if err := a.processOutboxEvent(ctx, event); err != nil {
			// Increment retry count (OutboxEventsMarkFailed handles status based on retry count)
			retryCount := event.RetryCount + 1
			errorMsg := err.Error()

			if markErr := a.storage.OutboxEventsMarkFailed(ctx, event.ID, retryCount, errorMsg); markErr != nil {
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
	switch event.EventType {
	case "account.saved":
		return a.publishAccountEvent(ctx, event)
	default:
		return fmt.Errorf("unknown event type: %s", event.EventType)
	}
}

func (a Activities) publishAccountEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal account event payload: %w", err)
	}

	// Create the event message (same format as EventsSendAccount)
	eventMessage := publish.EventMessage{
		IdempotencyKey: event.IdempotencyKey,
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedAccounts,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) deleteOutboxAndRecordSent(ctx context.Context, event models.OutboxEvent) error {
	// Record in events_sent table and delete from outbox in same transaction
	eventSent := models.EventSent{
		ID: models.EventID{
			EventIdempotencyKey: event.IdempotencyKey,
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
