package storage

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	defaultTasks = []models.Task{
		{
			ID: models.TaskID{
				Reference:   "1",
				ConnectorID: defaultConnector.ID,
			},
			ConnectorID: &defaultConnector.ID,
			Status:      "FAILED",
			CreatedAt:   now.Add(-time.Hour).UTC().Time,
			UpdatedAt:   now.Add(-time.Hour).UTC().Time,
			Error:       errors.New("test error"),
		},
		{
			ID: models.TaskID{
				Reference:   "2",
				ConnectorID: defaultConnector.ID,
			},
			ConnectorID: &defaultConnector.ID,
			Status:      "PROCESSING",
			CreatedAt:   now.Add(-2 * time.Hour).UTC().Time,
			UpdatedAt:   now.Add(-2 * time.Hour).UTC().Time,
		},
		{
			ID: models.TaskID{
				Reference:   "3",
				ConnectorID: defaultConnector.ID,
			},
			ConnectorID:     &defaultConnector.ID,
			Status:          "SUCCEEDED",
			CreatedAt:       now.Add(-3 * time.Hour).UTC().Time,
			UpdatedAt:       now.Add(-3 * time.Hour).UTC().Time,
			CreatedObjectID: pointer.For("test object id"),
		},
		{
			ID: models.TaskID{
				Reference:   "4",
				ConnectorID: defaultConnector2.ID,
			},
			ConnectorID: &defaultConnector2.ID,
			Status:      "PROCESSING",
			CreatedAt:   now.Add(-4 * time.Hour).UTC().Time,
			UpdatedAt:   now.Add(-4 * time.Hour).UTC().Time,
		},
	}
)

func upsertTasks(t *testing.T, ctx context.Context, storage Storage, tasks []models.Task) {
	t.Helper()

	for _, task := range tasks {
		err := storage.TasksUpsert(ctx, task)
		require.NoError(t, err)
	}
}

func TestTasksUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	upsertTasks(t, ctx, store, defaultTasks)

	t.Run("same id upsert", func(t *testing.T) {
		t2 := models.Task{
			ID:              defaultTasks[1].ID,
			ConnectorID:     &defaultConnector.ID,
			Status:          "FAILED",
			CreatedAt:       now.Add(-1 * time.Hour).UTC().Time,
			UpdatedAt:       now.Add(-1 * time.Hour).UTC().Time,
			CreatedObjectID: pointer.For("test object id 2"),
		}

		err := store.TasksUpsert(ctx, t2)
		require.NoError(t, err)

		t3, err := store.TasksGet(ctx, t2.ID)
		require.NoError(t, err)

		expectedTask := models.Task{
			ID:              defaultTasks[1].ID,
			ConnectorID:     &defaultConnector.ID,
			Status:          "FAILED",
			CreatedAt:       now.Add(-2 * time.Hour).UTC().Time,
			UpdatedAt:       now.Add(-1 * time.Hour).UTC().Time,
			CreatedObjectID: pointer.For("test object id 2"),
		}

		compareTasks(t, expectedTask, *t3)
	})

	t.Run("unknown connector id", func(t *testing.T) {
		t2 := models.Task{
			ID: models.TaskID{
				Reference: "5",
				ConnectorID: models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "unknown",
				},
			},
			ConnectorID: &models.ConnectorID{
				Reference: uuid.New(),
				Provider:  "unknown",
			},
			Status:    "FAILED",
			CreatedAt: now.Add(-1 * time.Hour).UTC().Time,
			UpdatedAt: now.Add(-1 * time.Hour).UTC().Time,
			Error:     errors.New("test error"),
		}

		err := store.TasksUpsert(ctx, t2)
		require.Error(t, err)
	})

	t.Run("outbox event created for task update", func(t *testing.T) {
		// Create a new task for this test
		newTask := models.Task{
			ID: models.TaskID{
				Reference:   "outbox-test-task",
				ConnectorID: defaultConnector.ID,
			},
			ConnectorID:     &defaultConnector.ID,
			Status:          "PROCESSING",
			CreatedAt:       now.Add(-10 * time.Minute).UTC().Time,
			UpdatedAt:       now.Add(-5 * time.Minute).UTC().Time,
			CreatedObjectID: pointer.For("test-object-id"),
		}

		expectedKey := newTask.IdempotencyKey()

		require.NoError(t, store.TasksUpsert(ctx, newTask))

		// Verify outbox event was created
		pendingEvents, err := store.OutboxEventsPollPending(ctx, 100)
		require.NoError(t, err)

		// Find our event
		var ourEvent *models.OutboxEvent
		for i := range pendingEvents {
			if pendingEvents[i].EventType == events.EventTypeUpdatedTask &&
				pendingEvents[i].EntityID == newTask.ID.String() &&
				pendingEvents[i].ID.EventIdempotencyKey == expectedKey {
				ourEvent = &pendingEvents[i]
				break
			}
		}
		require.NotNil(t, ourEvent, "expected outbox event for task updated")

		// Verify event details
		assert.Equal(t, events.EventTypeUpdatedTask, ourEvent.EventType)
		assert.Equal(t, models.OUTBOX_STATUS_PENDING, ourEvent.Status)
		assert.Equal(t, newTask.ID.String(), ourEvent.EntityID)
		assert.Equal(t, defaultConnector.ID, *ourEvent.ConnectorID)
		assert.Equal(t, 0, ourEvent.RetryCount)
		assert.Equal(t, expectedKey, ourEvent.ID.EventIdempotencyKey)

		// Verify payload
		var payload map[string]interface{}
		err = json.Unmarshal(ourEvent.Payload, &payload)
		require.NoError(t, err)
		assert.Equal(t, newTask.ID.String(), payload["id"])
		assert.Equal(t, string(newTask.Status), payload["status"])
		assert.NotNil(t, payload["createdAt"])
		assert.NotNil(t, payload["updatedAt"])
		if newTask.CreatedObjectID != nil {
			assert.Equal(t, *newTask.CreatedObjectID, payload["createdObjectID"])
		}
	})
}

func TestTasksGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	upsertTasks(t, ctx, store, defaultTasks)

	t.Run("get task", func(t *testing.T) {
		for _, task := range defaultTasks {
			t2, err := store.TasksGet(ctx, task.ID)
			require.NoError(t, err)
			compareTasks(t, task, *t2)
		}
	})

	t.Run("unknown task", func(t *testing.T) {
		_, err := store.TasksGet(ctx, models.TaskID{})
		require.ErrorIs(t, err, ErrNotFound)
		require.Error(t, err)
	})
}

func TestTasksDeleteFromConnectorID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	upsertTasks(t, ctx, store, defaultTasks)

	t.Run("unknown connector id", func(t *testing.T) {
		err := store.TasksDeleteFromConnectorID(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		})
		require.NoError(t, err)

		for _, task := range defaultTasks {
			t2, err := store.TasksGet(ctx, task.ID)
			require.NoError(t, err)
			compareTasks(t, task, *t2)
		}
	})

	t.Run("delete tasks", func(t *testing.T) {
		err := store.TasksDeleteFromConnectorID(ctx, defaultConnector.ID)
		require.NoError(t, err)

		for _, task := range defaultTasks {
			if task.ConnectorID != nil && *task.ConnectorID == defaultConnector.ID {
				_, err := store.TasksGet(ctx, task.ID)
				require.ErrorIs(t, err, ErrNotFound)
				require.Error(t, err)
			} else {
				t2, err := store.TasksGet(ctx, task.ID)
				require.NoError(t, err)
				compareTasks(t, task, *t2)
			}
		}
	})
}

func compareTasks(t *testing.T, expected, actual models.Task) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.ConnectorID, actual.ConnectorID)
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.UpdatedAt, actual.UpdatedAt)
	require.Equal(t, expected.CreatedObjectID, actual.CreatedObjectID)

	switch {
	case expected.Error == nil && actual.Error == nil:
	case expected.Error != nil && actual.Error != nil:
		require.Equal(t, expected.Error.Error(), actual.Error.Error())
	default:
		t.Errorf("expected error %v, got %v", expected.Error, actual.Error)
	}
}
