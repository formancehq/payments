package storage

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	internalErrors "github.com/formancehq/payments/internal/utils/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboxEventsInsert(t *testing.T) {
	store := newStore(t)
	defer store.Close()

	ctx := context.Background()

	// Insert a connector first
	upsertConnector(t, ctx, store, defaultConnector)

	// Create test events
	events := []models.OutboxEvent{
		{
			ID: models.EventID{
				EventIdempotencyKey: "test-idempotency-key-1",
				ConnectorID:         &defaultConnector.ID,
			},
			EventType:   "account.saved",
			EntityID:    "account-1",
			Payload:     json.RawMessage(`{"id": "account-1", "name": "Test Account"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
		},
		{
			ID: models.EventID{
				EventIdempotencyKey: "test-idempotency-key-2",
				ConnectorID:         &defaultConnector.ID,
			},
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
			ID: models.EventID{
				EventIdempotencyKey: "test-key-1",
				ConnectorID:         &defaultConnector.ID,
			},
		},
		{
			EventType:   "account.saved",
			EntityID:    "account-2",
			Payload:     json.RawMessage(`{"id": "account-2"}`),
			CreatedAt:   time.Now().UTC().Add(-1 * time.Minute), // Newer event
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
			ID: models.EventID{
				EventIdempotencyKey: "test-key-2",
				ConnectorID:         &defaultConnector.ID,
			},
		},
		{
			EventType:   "account.saved",
			EntityID:    "account-3",
			Payload:     json.RawMessage(`{"id": "account-3"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_FAILED, // Failed event should not be polled
			ConnectorID: &defaultConnector.ID,
			RetryCount:  1,
			ID: models.EventID{
				EventIdempotencyKey: "test-key-3",
				ConnectorID:         &defaultConnector.ID,
			},
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

// OutboxEventsDelete is no longer needed - marking as processed happens in OutboxEventsMarkProcessedAndRecordSent
// Keeping this test as a placeholder for future reference
func TestOutboxEventsMarkProcessedAndRecordSent(t *testing.T) {
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
			ID: models.EventID{
				EventIdempotencyKey: "test-key-for-delete-and-record",
				ConnectorID:         &defaultConnector.ID,
			},
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
		ID:          event.ID,
		ConnectorID: event.ConnectorID,
		SentAt:      time.Now().UTC(),
	}

	err = store.OutboxEventsMarkProcessedAndRecordSent(ctx, []models.EventID{event.ID}, []models.EventSent{eventSent})
	require.NoError(t, err)

	// Verify event is marked as processed (no longer pending)
	pendingEventsAfter, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEventsAfter, 0)

	// Verify event status is processed
	dbEvent := getOutboxEventByID(t, store, ctx, event.ID)
	assert.Equal(t, models.OUTBOX_STATUS_PROCESSED, dbEvent.Status)

	// Verify event is recorded as sent
	exists, err := store.EventsSentExists(ctx, event.ID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestOutboxEventsMarkProcessedAndRecordSent_Batch(t *testing.T) {
	store := newStore(t)
	defer store.Close()

	ctx := context.Background()

	// Insert a connector first
	upsertConnector(t, ctx, store, defaultConnector)

	// Insert test events
	events := []models.OutboxEvent{
		{
			EventType:   "account.saved",
			EntityID:    "account-1",
			Payload:     json.RawMessage(`{"id": "account-1"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
			ID: models.EventID{
				EventIdempotencyKey: "test-key-for-batch-1",
				ConnectorID:         &defaultConnector.ID,
			},
		},
		{
			EventType:   "account.saved",
			EntityID:    "account-2",
			Payload:     json.RawMessage(`{"id": "account-2"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
			ID: models.EventID{
				EventIdempotencyKey: "test-key-for-batch-2",
				ConnectorID:         &defaultConnector.ID,
			},
		},
		{
			EventType:   "account.saved",
			EntityID:    "account-3",
			Payload:     json.RawMessage(`{"id": "account-3"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
			ID: models.EventID{
				EventIdempotencyKey: "test-key-for-batch-3",
				ConnectorID:         &defaultConnector.ID,
			},
		},
	}

	insertOutboxEventsWithTx(t, store, ctx, events)

	// Get the events
	pendingEvents, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, pendingEvents, 3)

	// Prepare batch delete and record sent
	eventIDs := make([]models.EventID, 0, len(pendingEvents))
	eventsSent := make([]models.EventSent, 0, len(pendingEvents))
	for _, event := range pendingEvents {
		eventIDs = append(eventIDs, event.ID)
		eventsSent = append(eventsSent, models.EventSent{
			ID:          event.ID,
			ConnectorID: event.ConnectorID,
			SentAt:      time.Now().UTC(),
		})
	}

	// Batch mark as processed and record sent
	err = store.OutboxEventsMarkProcessedAndRecordSent(ctx, eventIDs, eventsSent)
	require.NoError(t, err)

	// Verify all events are marked as processed (no longer pending)
	pendingEventsAfter, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEventsAfter, 0)

	// Verify all events are recorded as sent
	for _, event := range pendingEvents {
		exists, err := store.EventsSentExists(ctx, event.ID)
		require.NoError(t, err)
		assert.True(t, exists, "Event %s should be recorded as sent", event.ID.EventIdempotencyKey)

		// Verify event status is processed
		dbEvent := getOutboxEventByID(t, store, ctx, event.ID)
		assert.Equal(t, models.OUTBOX_STATUS_PROCESSED, dbEvent.Status)
	}
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
			ID: models.EventID{
				EventIdempotencyKey: "test-key-for-mark-failed",
				ConnectorID:         &defaultConnector.ID,
			},
		},
	}

	insertOutboxEventsWithTx(t, store, ctx, events)

	// Get the event
	pendingEvents, err := store.OutboxEventsPollPending(ctx, 1)
	require.NoError(t, err)
	require.Len(t, pendingEvents, 1)
	event := pendingEvents[0]

	// Mark as failed with retry count less than max (should remain PENDING for retry)
	retryCount := 1
	testErr := errors.New("test error")
	err = store.OutboxEventsMarkFailed(ctx, event.ID, retryCount, testErr)
	require.NoError(t, err)

	// Verify event is still pending (not yet at max retries)
	pendingEventsAfter, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEventsAfter, 1, "Event should still be pending for retry")
	assert.Equal(t, retryCount, pendingEventsAfter[0].RetryCount)
	assert.Equal(t, models.OUTBOX_STATUS_PENDING, pendingEventsAfter[0].Status)
	assert.NotNil(t, pendingEventsAfter[0].Error)
	assert.Equal(t, testErr.Error(), *pendingEventsAfter[0].Error)

	// Now test with max retries exceeded (should move to FAILED)
	err = store.OutboxEventsMarkFailed(ctx, event.ID, models.MaxOutboxRetries, testErr)
	require.NoError(t, err)

	// Verify event is no longer pending (marked as FAILED)
	pendingEventsAfterFailed, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEventsAfterFailed, 0, "Event should be marked as FAILED")
}

func TestOutboxEventsMarkFailed_NonRetryableError(t *testing.T) {
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
			ID: models.EventID{
				EventIdempotencyKey: "test-key-for-non-retryable",
				ConnectorID:         &defaultConnector.ID,
			},
		},
	}

	insertOutboxEventsWithTx(t, store, ctx, events)

	// Get the event
	pendingEvents, err := store.OutboxEventsPollPending(ctx, 1)
	require.NoError(t, err)
	require.Len(t, pendingEvents, 1)
	event := pendingEvents[0]

	// Mark as failed with a non-retryable error and low retry count
	// Should immediately move to FAILED status regardless of retry count
	retryCount := 1
	nonRetryableErr := &testNonRetryableError{message: "non-retryable validation error"}

	// Verify the error implements the interface
	var _ internalErrors.NonRetryableError = nonRetryableErr

	err = store.OutboxEventsMarkFailed(ctx, event.ID, retryCount, nonRetryableErr)
	require.NoError(t, err)

	// Verify event is immediately marked as FAILED (not pending for retry)
	pendingEventsAfter, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEventsAfter, 0, "Event should be immediately marked as FAILED for non-retryable error")

	// Verify the event status is FAILED by querying directly
	// We need to check the status in the database since it's no longer pending
	dbEvent := getOutboxEventByID(t, store, ctx, event.ID)
	assert.Equal(t, models.OUTBOX_STATUS_FAILED, dbEvent.Status)
	assert.Equal(t, retryCount, dbEvent.RetryCount)
	assert.NotNil(t, dbEvent.Error)
	assert.Equal(t, nonRetryableErr.Error(), *dbEvent.Error)
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

func TestOutboxEventsInsert_FiltersEventsAlreadySent(t *testing.T) {
	store := newStore(t)
	defer store.Close()

	ctx := context.Background()

	// Insert connectors
	upsertConnector(t, ctx, store, defaultConnector)

	// Create an event that will be marked as sent
	eventSent := models.EventSent{
		ID: models.EventID{
			EventIdempotencyKey: "already-sent-key",
			ConnectorID:         &defaultConnector.ID,
		},
		ConnectorID: &defaultConnector.ID,
		SentAt:      time.Now().UTC(),
	}

	// Record the event as sent
	err := store.EventsSentUpsert(ctx, eventSent)
	require.NoError(t, err)

	// Try to insert events, including one with the same idempotency key that was already sent
	events := []models.OutboxEvent{
		{
			EventType:   "account.saved",
			EntityID:    "account-1",
			Payload:     json.RawMessage(`{"id": "account-1"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
			ID: models.EventID{
				EventIdempotencyKey: "already-sent-key",
				ConnectorID:         &defaultConnector.ID,
			}, // This should be filtered out
		},
		{
			EventType:   "account.saved",
			EntityID:    "account-2",
			Payload:     json.RawMessage(`{"id": "account-2"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
			ID: models.EventID{
				EventIdempotencyKey: "new-key",
				ConnectorID:         &defaultConnector.ID,
			}, // This should be inserted
		},
	}

	insertOutboxEventsWithTx(t, store, ctx, events)

	// Verify only the new event was inserted
	pendingEvents, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEvents, 1)
	assert.Equal(t, "new-key", pendingEvents[0].ID.EventIdempotencyKey)
	assert.Equal(t, "account-2", pendingEvents[0].EntityID)
}

func TestOutboxEventsInsert_UniqueConstraintOnIdempotencyKeyAndConnectorID(t *testing.T) {
	store := newStore(t)
	defer store.Close()

	ctx := context.Background()

	// Insert connectors
	upsertConnector(t, ctx, store, defaultConnector)

	// Insert an event with a specific idempotency key and connector
	events1 := []models.OutboxEvent{
		{
			EventType:   "account.saved",
			EntityID:    "account-1",
			Payload:     json.RawMessage(`{"id": "account-1"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
			ID: models.EventID{
				EventIdempotencyKey: "duplicate-key",
				ConnectorID:         &defaultConnector.ID,
			},
		},
	}

	insertOutboxEventsWithTx(t, store, ctx, events1)

	// Try to insert another event with the same idempotency key and connector
	// This should be handled by ON CONFLICT DO NOTHING
	events2 := []models.OutboxEvent{
		{
			EventType:   "account.saved",
			EntityID:    "account-2", // Different entity
			Payload:     json.RawMessage(`{"id": "account-2"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID, // Same connector
			RetryCount:  0,
			ID: models.EventID{
				EventIdempotencyKey: "duplicate-key",
				ConnectorID:         &defaultConnector.ID,
			}, // Same idempotency key
		},
	}

	// This should not error due to ON CONFLICT DO NOTHING
	insertOutboxEventsWithTx(t, store, ctx, events2)

	// Verify only one event exists (the first one)
	pendingEvents, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEvents, 1)
	assert.Equal(t, "account-1", pendingEvents[0].EntityID)
	assert.Equal(t, "duplicate-key", pendingEvents[0].ID.EventIdempotencyKey)
}

func TestOutboxEventsInsert_SameIdempotencyKeyDifferentConnector(t *testing.T) {
	store := newStore(t)
	defer store.Close()

	ctx := context.Background()

	// Insert connectors
	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)

	// Insert an event with a specific idempotency key and connector
	events1 := []models.OutboxEvent{
		{
			ID: models.EventID{
				EventIdempotencyKey: "shared-key",
				ConnectorID:         &defaultConnector.ID,
			},
			EventType:   "account.saved",
			EntityID:    "account-1",
			Payload:     json.RawMessage(`{"id": "account-1"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
		},
	}

	insertOutboxEventsWithTx(t, store, ctx, events1)

	// Insert another event with the same idempotency key but different connector
	// This should succeed because the unique constraint is on (idempotency_key, connector_id)
	events2 := []models.OutboxEvent{
		{
			ID: models.EventID{
				EventIdempotencyKey: "shared-key",
				ConnectorID:         &defaultConnector2.ID,
			},
			EventType:   "account.saved",
			EntityID:    "account-2",
			Payload:     json.RawMessage(`{"id": "account-2"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector2.ID, // Different connector
			RetryCount:  0,
		},
	}

	insertOutboxEventsWithTx(t, store, ctx, events2)

	// Verify both events exist
	pendingEvents, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEvents, 2)

	// Verify both have the same idempotency key but different connectors
	foundConnector1 := false
	foundConnector2 := false
	for _, event := range pendingEvents {
		assert.Equal(t, "shared-key", event.ID.EventIdempotencyKey)
		if event.ConnectorID != nil {
			if *event.ConnectorID == defaultConnector.ID {
				foundConnector1 = true
			} else if *event.ConnectorID == defaultConnector2.ID {
				foundConnector2 = true
			}
		}
	}
	assert.True(t, foundConnector1, "Should have event with defaultConnector")
	assert.True(t, foundConnector2, "Should have event with defaultConnector2")
}

func TestOutboxEventsInsert_SameIdempotencyKeyWithNilConnectorID(t *testing.T) {
	store := newStore(t)
	defer store.Close()

	ctx := context.Background()

	// Insert a connector
	upsertConnector(t, ctx, store, defaultConnector)

	// Insert an event with a specific idempotency key and connector
	events1 := []models.OutboxEvent{
		{
			ID: models.EventID{
				EventIdempotencyKey: "nil-connector-test",
				ConnectorID:         &defaultConnector.ID,
			},
			EventType:   "account.saved",
			EntityID:    "account-1",
			Payload:     json.RawMessage(`{"id": "account-1"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: &defaultConnector.ID,
			RetryCount:  0,
		},
	}

	insertOutboxEventsWithTx(t, store, ctx, events1)

	// Insert another event with the same idempotency key but nil connector_id
	// In PostgreSQL, NULL values are considered distinct in unique constraints,
	// so this should succeed
	events2 := []models.OutboxEvent{
		{
			EventType:   "account.saved",
			EntityID:    "account-2",
			Payload:     json.RawMessage(`{"id": "account-2"}`),
			CreatedAt:   time.Now().UTC(),
			Status:      models.OUTBOX_STATUS_PENDING,
			ConnectorID: nil, // Nil connector
			RetryCount:  0,
			ID: models.EventID{
				EventIdempotencyKey: "nil-connector-test",
				ConnectorID:         nil,
			}, // Same idempotency key
		},
	}

	insertOutboxEventsWithTx(t, store, ctx, events2)

	// Verify both events exist (NULL and non-NULL connector_id are distinct)
	pendingEvents, err := store.OutboxEventsPollPending(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, pendingEvents, 2)

	// Verify one has connector_id and one doesn't
	hasConnector := false
	hasNilConnector := false
	for _, event := range pendingEvents {
		if event.ConnectorID != nil {
			hasConnector = true
		} else {
			hasNilConnector = true
		}
	}
	assert.True(t, hasConnector)
	assert.True(t, hasNilConnector)
}

// testNonRetryableError is a test helper that implements NonRetryableError interface
type testNonRetryableError struct {
	message string
}

func (e *testNonRetryableError) Error() string {
	return e.message
}

func (e *testNonRetryableError) NonRetryable() {}

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

// Helper function to get outbox event by ID for testing
func getOutboxEventByID(t *testing.T, s Storage, ctx context.Context, eventID models.EventID) outboxEvent {
	storeImpl := s.(*store)
	var dbEvent outboxEvent
	err := storeImpl.db.NewSelect().
		TableExpr("outbox_events").
		Where("id = ?", eventID).
		Scan(ctx, &dbEvent)
	require.NoError(t, err)
	return dbEvent
}
