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
	case models.OUTBOX_EVENT_ACCOUNT_SAVED:
		return a.publishAccountEvent(ctx, event)
	case models.OUTBOX_EVENT_BALANCE_SAVED:
		return a.publishBalanceEvent(ctx, event)
	case models.OUTBOX_EVENT_PAYMENT_SAVED:
		return a.publishPaymentEvent(ctx, event)
	case models.OUTBOX_EVENT_PAYMENT_DELETED:
		return a.publishPaymentDeletedEvent(ctx, event)
	case models.OUTBOX_EVENT_BANK_ACCOUNT_SAVED:
		return a.publishBankAccountEvent(ctx, event)
	case models.OUTBOX_EVENT_TASK_UPDATED:
		return a.publishTaskEvent(ctx, event)
	case models.OUTBOX_EVENT_CONNECTOR_RESET:
		return a.publishConnectorResetEvent(ctx, event)
	case models.OUTBOX_EVENT_POOL_SAVED:
		return a.publishPoolSavedEvent(ctx, event)
	case models.OUTBOX_EVENT_POOL_DELETED:
		return a.publishPoolDeletedEvent(ctx, event)
	case models.OUTBOX_EVENT_PAYMENT_INITIATION_SAVED:
		return a.publishPaymentInitiationEvent(ctx, event)
	case models.OUTBOX_EVENT_PAYMENT_INITIATION_ADJUSTMENT_SAVED:
		return a.publishPaymentInitiationAdjustmentEvent(ctx, event)
	case models.OUTBOX_EVENT_PAYMENT_INITIATION_RELATED_PAYMENT_SAVED:
		return a.publishPaymentInitiationRelatedPaymentEvent(ctx, event)
	case models.OUTBOX_EVENT_USER_LINK_STATUS:
		return a.publishUserLinkStatusEvent(ctx, event)
	case models.OUTBOX_EVENT_USER_CONNECTION_DATA_SYNCED:
		return a.publishUserConnectionDataSyncedEvent(ctx, event)
	case models.OUTBOX_EVENT_USER_CONNECTION_PENDING_DISCONNECT:
		return a.publishUserConnectionPendingDisconnectEvent(ctx, event)
	case models.OUTBOX_EVENT_USER_CONNECTION_DISCONNECTED:
		return a.publishUserConnectionDisconnectedEvent(ctx, event)
	case models.OUTBOX_EVENT_USER_CONNECTION_RECONNECTED:
		return a.publishUserConnectionReconnectedEvent(ctx, event)
	case models.OUTBOX_EVENT_USER_DISCONNECTED:
		return a.publishUserDisconnectedEvent(ctx, event)
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
		IdempotencyKey: a.generateIdempotencyKey(event), // TODO when don't we have an indepeotency key?
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedAccounts,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishBalanceEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal balance event payload: %w", err)
	}

	// Create the event message (same format as EventsSendBalance)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedBalances,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishPaymentEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payment event payload: %w", err)
	}

	// Create the event message (same format as EventsSendPayment)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedPayments,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishPaymentDeletedEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payment deleted event payload: %w", err)
	}

	// Create the event message (same format as EventsSendPaymentDeleted)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeDeletedPayments,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishBankAccountEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal bank account event payload: %w", err)
	}

	// Create the event message (same format as EventsSendBankAccount)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedBankAccount,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishTaskEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal task event payload: %w", err)
	}

	// Create the event message (same format as EventsSendTaskUpdated)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeUpdatedTask,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishConnectorResetEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal connector reset event payload: %w", err)
	}

	// Create the event message (same format as EventsSendConnectorReset)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeConnectorReset,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishPoolSavedEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal pool saved event payload: %w", err)
	}

	// Create the event message (same format as EventsSendPoolCreation)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedPool,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishPoolDeletedEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal pool deleted event payload: %w", err)
	}

	// Create the event message (same format as EventsSendPoolDeletion)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeDeletePool,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishPaymentInitiationEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payment initiation event payload: %w", err)
	}

	// Create the event message (same format as EventsSendPaymentInitiation)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedPaymentInitiation,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishPaymentInitiationAdjustmentEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payment initiation adjustment event payload: %w", err)
	}

	// Create the event message (same format as EventsSendPaymentInitiationAdjustment)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedPaymentInitiationAdjustment,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishPaymentInitiationRelatedPaymentEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payment initiation related payment event payload: %w", err)
	}

	// Create the event message (same format as EventsSendPaymentInitiationRelatedPayment)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedPaymentInitiationRelatedPayment,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishUserLinkStatusEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal user link status event payload: %w", err)
	}

	// Create the event message (same format as EventsSendUserLinkStatus)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeOpenBankingUserLinkStatus,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishUserConnectionDataSyncedEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal user connection data synced event payload: %w", err)
	}

	// Create the event message (same format as EventsSendUserConnectionDataSynced)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeOpenBankingUserConnectionDataSynced,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishUserConnectionPendingDisconnectEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal user connection pending disconnect event payload: %w", err)
	}

	// Create the event message (same format as EventsSendUserPendingDisconnect)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeOpenBankingUserConnectionPendingDisconnect,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishUserConnectionDisconnectedEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal user connection disconnected event payload: %w", err)
	}

	// Create the event message (same format as EventsSendUserConnectionDisconnected)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeOpenBankingUserConnectionDisconnected,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishUserConnectionReconnectedEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal user connection reconnected event payload: %w", err)
	}

	// Create the event message (same format as EventsSendUserConnectionReconnected)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeOpenBankingUserConnectionReconnected,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) publishUserDisconnectedEvent(ctx context.Context, event models.OutboxEvent) error {
	// Parse the payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal user disconnected event payload: %w", err)
	}

	// Create the event message (same format as EventsSendUserDisconnected)
	eventMessage := publish.EventMessage{
		IdempotencyKey: a.generateIdempotencyKey(event),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeOpenBankingUserDisconnected,
		Payload:        payload,
	}

	// Publish the event
	return a.events.Publish(ctx, eventMessage)
}

func (a Activities) generateIdempotencyKey(event models.OutboxEvent) string {
	if event.IdempotencyKey != "" {
		return event.IdempotencyKey
	}
	// Fallback: generate idempotency key based on event type and entity ID
	return fmt.Sprintf("%s:%s", event.EventType, event.EntityID)
}

func (a Activities) deleteOutboxAndRecordSent(ctx context.Context, event models.OutboxEvent) error {
	// Record in events_sent table and delete from outbox in same transaction
	idempotencyKey := a.generateIdempotencyKey(event) // todo when don't we have an idempotencyKey?
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
