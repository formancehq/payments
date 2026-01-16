package storage

import (
	"context"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var (
	defaultEventsSent = []models.EventSent{
		{
			ID: models.EventID{
				EventIdempotencyKey: "test1",
				ConnectorID:         &defaultConnector.ID,
			},
			ConnectorID: &defaultConnector.ID,
			SentAt:      now.UTC().Time,
		},
		{
			ID: models.EventID{
				EventIdempotencyKey: "test2",
				ConnectorID:         &defaultConnector.ID,
			},
			ConnectorID: &defaultConnector.ID,
			SentAt:      now.Add(-1 * time.Hour).UTC().Time,
		},
		{
			ID: models.EventID{
				EventIdempotencyKey: "test3",
				ConnectorID:         &defaultConnector2.ID,
			},
			ConnectorID: &defaultConnector2.ID,
			SentAt:      now.Add(-2 * time.Hour).UTC().Time,
		},
		{
			ID: models.EventID{
				EventIdempotencyKey: "test4",
				ConnectorID:         nil,
			},
			ConnectorID: nil,
			SentAt:      now.Add(-3 * time.Hour).UTC().Time,
		},
	}
)

func upsertEventsSent(t *testing.T, ctx context.Context, storage Storage, eventsSent []models.EventSent) {
	for _, e := range eventsSent {
		require.NoError(t, storage.EventsSentUpsert(ctx, e))
	}
}

func TestEventsSentUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	upsertEventsSent(t, ctx, store, defaultEventsSent)

	t.Run("same id insert", func(t *testing.T) {
		id := models.EventID{
			EventIdempotencyKey: "test1",
			ConnectorID:         &defaultConnector.ID,
		}

		e := models.EventSent{
			ID:          id,
			ConnectorID: &defaultConnector.ID,
			SentAt:      now.Add(-3 * time.Hour).UTC().Time, // changed
		}

		require.NoError(t, store.EventsSentUpsert(ctx, e))

		got, err := store.EventsSentGet(ctx, id)
		require.NoError(t, err)
		require.Equal(t, defaultEventsSent[0], *got)
	})

	t.Run("unknown connector id", func(t *testing.T) {
		unknownConnectorID := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}

		e := models.EventSent{
			ID: models.EventID{
				EventIdempotencyKey: "test5",
				ConnectorID:         &unknownConnectorID,
			},
			ConnectorID: &unknownConnectorID,
			SentAt:      now.Add(-3 * time.Hour).UTC().Time,
		}

		require.Error(t, store.EventsSentUpsert(ctx, e))
	})
}

func TestEventsSentGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	upsertEventsSent(t, ctx, store, defaultEventsSent)

	t.Run("get event sent", func(t *testing.T) {
		for _, e := range defaultEventsSent {
			got, err := store.EventsSentGet(ctx, e.ID)
			require.NoError(t, err)
			require.Equal(t, e, *got)
		}
	})

	t.Run("unknown event sent", func(t *testing.T) {
		got, err := store.EventsSentGet(ctx, models.EventID{
			EventIdempotencyKey: "unknown",
			ConnectorID:         &defaultConnector.ID,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, got)
	})
}

func TestEventsSentExist(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	upsertEventsSent(t, ctx, store, defaultEventsSent)

	t.Run("existing", func(t *testing.T) {
		for _, e := range defaultEventsSent {
			got, err := store.EventsSentExists(ctx, e.ID)
			require.NoError(t, err)
			require.Equal(t, true, got)
		}
	})

	t.Run("not existing", func(t *testing.T) {
		got, err := store.EventsSentExists(ctx, models.EventID{
			EventIdempotencyKey: "unknown",
			ConnectorID:         &defaultConnector.ID,
		})
		require.NoError(t, err)
		require.Equal(t, false, got)
	})
}

func TestEventsSentDelete(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	upsertEventsSent(t, ctx, store, defaultEventsSent)

	t.Run("delete from unknown connector id", func(t *testing.T) {
		unknownConnectorID := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}

		require.NoError(t, store.EventsSentDeleteFromConnectorID(ctx, unknownConnectorID))

		for _, e := range defaultEventsSent {
			got, err := store.EventsSentGet(ctx, e.ID)
			require.NoError(t, err)
			require.Equal(t, e, *got)
		}
	})

	t.Run("delete from connector id", func(t *testing.T) {
		require.NoError(t, store.EventsSentDeleteFromConnectorID(ctx, defaultConnector.ID))

		for _, e := range defaultEventsSent {
			if e.ConnectorID != nil && *e.ConnectorID == defaultConnector.ID {
				got, err := store.EventsSentGet(ctx, e.ID)
				require.Error(t, err)
				require.ErrorIs(t, err, ErrNotFound)
				require.Nil(t, got)
			} else {
				got, err := store.EventsSentGet(ctx, e.ID)
				require.NoError(t, err)
				require.Equal(t, e, *got)
			}
		}
	})
}

func TestEventsSentDeleteFromConnectorIDBatch(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)

	t.Run("invalid batchSize zero", func(t *testing.T) {
		_, err := store.EventsSentDeleteFromConnectorIDBatch(ctx, defaultConnector.ID, 0)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidation)
		require.Contains(t, err.Error(), "invalid batchSize 0")
		require.Contains(t, err.Error(), defaultConnector.ID.String())
	})

	t.Run("invalid batchSize negative", func(t *testing.T) {
		_, err := store.EventsSentDeleteFromConnectorIDBatch(ctx, defaultConnector.ID, -1)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidation)
		require.Contains(t, err.Error(), "invalid batchSize -1")
		require.Contains(t, err.Error(), defaultConnector.ID.String())
	})

	t.Run("delete batch from unknown connector", func(t *testing.T) {
		rowsAffected, err := store.EventsSentDeleteFromConnectorIDBatch(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}, 10)
		require.NoError(t, err)
		require.Equal(t, 0, rowsAffected)
	})

	t.Run("delete batch with no events", func(t *testing.T) {
		rowsAffected, err := store.EventsSentDeleteFromConnectorIDBatch(ctx, defaultConnector.ID, 10)
		require.NoError(t, err)
		require.Equal(t, 0, rowsAffected)
	})

	t.Run("delete single batch smaller than batch size", func(t *testing.T) {
		// Insert events for defaultConnector
		events := []models.EventSent{
			{
				ID: models.EventID{
					EventIdempotencyKey: "batch-test-1",
					ConnectorID:         &defaultConnector.ID,
				},
				ConnectorID: &defaultConnector.ID,
				SentAt:      now.UTC().Time,
			},
			{
				ID: models.EventID{
					EventIdempotencyKey: "batch-test-2",
					ConnectorID:         &defaultConnector.ID,
				},
				ConnectorID: &defaultConnector.ID,
				SentAt:      now.Add(-1 * time.Hour).UTC().Time,
			},
		}
		upsertEventsSent(t, ctx, store, events)

		rowsAffected, err := store.EventsSentDeleteFromConnectorIDBatch(ctx, defaultConnector.ID, 10)
		require.NoError(t, err)
		require.Equal(t, 2, rowsAffected)

		// Verify all events were deleted
		for _, e := range events {
			_, err := store.EventsSentGet(ctx, e.ID)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrNotFound)
		}
	})

	t.Run("delete batch only affects specified connector", func(t *testing.T) {
		// Insert events for both connectors
		eventsConnector1 := []models.EventSent{
			{
				ID: models.EventID{
					EventIdempotencyKey: "isolation-test-1",
					ConnectorID:         &defaultConnector.ID,
				},
				ConnectorID: &defaultConnector.ID,
				SentAt:      now.UTC().Time,
			},
		}
		eventsConnector2 := []models.EventSent{
			{
				ID: models.EventID{
					EventIdempotencyKey: "isolation-test-2",
					ConnectorID:         &defaultConnector2.ID,
				},
				ConnectorID: &defaultConnector2.ID,
				SentAt:      now.UTC().Time,
			},
		}
		upsertEventsSent(t, ctx, store, eventsConnector1)
		upsertEventsSent(t, ctx, store, eventsConnector2)

		// Delete from connector1
		rowsAffected, err := store.EventsSentDeleteFromConnectorIDBatch(ctx, defaultConnector.ID, 10)
		require.NoError(t, err)
		require.Equal(t, 1, rowsAffected)

		// Verify connector1 events are deleted
		_, err = store.EventsSentGet(ctx, eventsConnector1[0].ID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)

		// Verify connector2 events still exist
		got, err := store.EventsSentGet(ctx, eventsConnector2[0].ID)
		require.NoError(t, err)
		require.Equal(t, eventsConnector2[0], *got)
	})
}
