package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to insert outbox events within a transaction
func insertOutboxEventsWithTx(t *testing.T, s Storage, ctx context.Context, events []models.OutboxEvent) {
	// Type assert to *store to access db field
	store := s.(*store)

	tx, err := store.db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	err = s.OutboxEventsInsert(ctx, tx, events)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)
}

func TestOutboxEventsInsert(t *testing.T) {
	store := newStore(t)
	defer store.Close()

	ctx := context.Background()

	// Insert a connector first
	upsertConnector(t, ctx, store, defaultConnector)

	// Create test events
	events := []models.OutboxEvent{
		{
			EventType:   "account.saved",
			EntityID:    "account-1",
			Payload:     json.RawMessage(`{"id": "account-1", "name": "Test Account"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
		},
		{
			EventType:   "account.saved",
			EntityID:    "account-2",
			Payload:     json.RawMessage(`{"id": "account-2", "name": "Test Account 2"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
		},
	}

	// Insert events within a transaction
	insertOutboxEventsWithTx(t, store, ctx, events)

	// Verify events were inserted
	pendingEvents, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEvents, 2)

	// Check event details
	assert.Equal(t, "account.saved", pendingEvents[0].EventType)
	assert.Equal(t, "account-1", pendingEvents[0].EntityID)
	assert.Equal(t, models.OUTBOX_STATUS_PENDING, pendingEvents[0].Status)
	assert.Equal(t, 0, pendingEvents[0].RetryCount)
}

func TestOutboxEventsPollPending(t *testing.T) {
	store := newStore(t)
	defer store.Close()

	ctx := context.Background()

	// Insert a connector first
	upsertConnector(t, ctx, store, defaultConnector)

	// Insert test events with different statuses
	events := []models.OutboxEvent{
		{
			EventType:   "account.saved",
			EntityID:    "account-1",
			Payload:     json.RawMessage(`{"id": "account-1"}`),
			CreatedAt:   time.Now().UTC().Add(-2 * time.Minute), // Older event
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
		},
		{
			EventType:   "account.saved",
			EntityID:    "account-2",
			Payload:     json.RawMessage(`{"id": "account-2"}`),
			CreatedAt:   time.Now().UTC().Add(-1 * time.Minute), // Newer event
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
		},
		{
			EventType:   "account.saved",
			EntityID:    "account-3",
			Payload:     json.RawMessage(`{"id": "account-3"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_FAILED, // Failed event should not be polled
			ConnectorID: &defaultConnector.ID,
			RetryCount:  1,
		},
	}

	insertOutboxEventsWithTx(t, store, ctx, events)

	// Poll pending events
	pendingEvents, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)

	// Should only return pending events, ordered by created_at ASC
	assert.Len(t, pendingEvents, 2)
	assert.Equal(t, "account-1", pendingEvents[0].EntityID) // Older event first
	assert.Equal(t, "account-2", pendingEvents[1].EntityID) // Newer event second
	assert.Equal(t, models.OUTBOX_STATUS_PENDING, pendingEvents[0].Status)
	assert.Equal(t, models.OUTBOX_STATUS_PENDING, pendingEvents[1].Status)

	// Test limit
	pendingEventsLimited, err := store.OutboxEventsPollPending(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, pendingEventsLimited, 1)
	assert.Equal(t, "account-1", pendingEventsLimited[0].EntityID) // Should get the oldest
}

// OutboxEventsDelete is no longer needed - deletion happens in OutboxEventsDeleteAndRecordSent
// Keeping this test as a placeholder for future reference
func TestOutboxEventsDeleteAndRecordSent(t *testing.T) {
	store := newStore(t)
	defer store.Close()

	ctx := context.Background()

	// Insert a connector first
	upsertConnector(t, ctx, store, defaultConnector)

	// Insert test event
	events := []models.OutboxEvent{
		{
			EventType:   "account.saved",
			EntityID:    "account-1",
			Payload:     json.RawMessage(`{"id": "account-1"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
		},
	}

	insertOutboxEventsWithTx(t, store, ctx, events)

	// Get the event
	pendingEvents, err := store.OutboxEventsPollPending(ctx, 1)
	require.NoError(t, err)
	require.Len(t, pendingEvents, 1)
	event := pendingEvents[0]

	// Delete and record sent in transaction
	eventSent := models.EventSent{
		ID: models.EventID{
			EventIdempotencyKey: fmt.Sprintf("%s:%s", event.EventType, event.EntityID),
			ConnectorID:         event.ConnectorID,
		},
		ConnectorID: event.ConnectorID,
		SentAt:      time.Now().UTC(),
	}

	err = store.OutboxEventsDeleteAndRecordSent(ctx, event.ID, eventSent)
	require.NoError(t, err)

	// Verify event is deleted
	pendingEventsAfter, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEventsAfter, 0)

	// Verify event is recorded as sent
	eventID := models.EventID{
		EventIdempotencyKey: fmt.Sprintf("%s:%s", event.EventType, event.EntityID),
		ConnectorID:         event.ConnectorID,
	}
	exists, err := store.EventsSentExists(ctx, eventID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestOutboxEventsMarkFailed(t *testing.T) {
	store := newStore(t)
	defer store.Close()

	ctx := context.Background()

	// Insert a connector first
	upsertConnector(t, ctx, store, defaultConnector)

	// Insert test event
	events := []models.OutboxEvent{
		{
			EventType:   "account.saved",
			EntityID:    "account-1",
			Payload:     json.RawMessage(`{"id": "account-1"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
		},
	}

	insertOutboxEventsWithTx(t, store, ctx, events)

	// Get the event ID
	pendingEvents, err := store.OutboxEventsPollPending(ctx, 1)
	require.NoError(t, err)
	require.Len(t, pendingEvents, 1)
	eventID := pendingEvents[0].ID

	// Mark as failed with retry count less than max (should remain PENDING for retry)
	retryCount := 1
	errorMsg := "test error"
	err = store.OutboxEventsMarkFailed(ctx, eventID, retryCount, errorMsg)
	require.NoError(t, err)

	// Verify event is still pending (not yet at max retries)
	pendingEventsAfter, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEventsAfter, 1, "Event should still be pending for retry")
	assert.Equal(t, retryCount, pendingEventsAfter[0].RetryCount)
	assert.Equal(t, models.OUTBOX_STATUS_PENDING, pendingEventsAfter[0].Status)
	assert.NotNil(t, pendingEventsAfter[0].Error)
	assert.Equal(t, errorMsg, *pendingEventsAfter[0].Error)

	// Now test with max retries exceeded (should move to FAILED)
	err = store.OutboxEventsMarkFailed(ctx, eventID, models.MaxOutboxRetries, errorMsg)
	require.NoError(t, err)

	// Verify event is no longer pending (marked as FAILED)
	pendingEventsAfterFailed, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEventsAfterFailed, 0, "Event should be marked as FAILED")
}

func TestOutboxEventsEmptyInsert(t *testing.T) {
	store := newStore(t)
	defer store.Close()

	ctx := context.Background()

	// Insert empty slice should not error - using the helper with empty slice
	insertOutboxEventsWithTx(t, store, ctx, []models.OutboxEvent{})

	// Poll should return empty
	pendingEvents, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEvents, 0)
}
