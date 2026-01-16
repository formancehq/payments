package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var (
	defaultWorkflowInstances = []models.Instance{
		{
			ID:          "test1",
			ScheduleID:  defaultSchedules[0].ID,
			ConnectorID: defaultConnector.ID,
			CreatedAt:   now.Add(-60 * time.Minute).UTC().Time,
			UpdatedAt:   now.Add(-60 * time.Minute).UTC().Time,
			Terminated:  false,
		},
		{
			ID:          "test2",
			ScheduleID:  defaultSchedules[0].ID,
			ConnectorID: defaultConnector.ID,
			CreatedAt:   now.Add(-30 * time.Minute).UTC().Time,
			UpdatedAt:   now.Add(-30 * time.Minute).UTC().Time,
			Terminated:  false,
		},
		{
			ID:           "test3",
			ScheduleID:   defaultSchedules[2].ID,
			ConnectorID:  defaultConnector.ID,
			CreatedAt:    now.Add(-55 * time.Minute).UTC().Time,
			UpdatedAt:    now.Add(-55 * time.Minute).UTC().Time,
			Terminated:   true,
			TerminatedAt: pointer.For(now.UTC().Time),
			Error:        pointer.For("test error"),
		},
	}
)

func upsertInstance(t *testing.T, ctx context.Context, storage Storage, instance models.Instance) {
	require.NoError(t, storage.InstancesUpsert(ctx, instance))
}

func TestInstancesUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	for _, schedule := range defaultSchedules {
		upsertSchedule(t, ctx, store, schedule)
	}
	for _, instance := range defaultWorkflowInstances {
		upsertInstance(t, ctx, store, instance)
	}

	t.Run("same id upsert", func(t *testing.T) {
		instance := defaultWorkflowInstances[0]
		instance.Terminated = true
		instance.TerminatedAt = pointer.For(now.UTC().Time)
		instance.Error = pointer.For("test error")

		upsertInstance(t, ctx, store, instance)

		actual, err := store.InstancesGet(ctx, instance.ID, instance.ScheduleID, instance.ConnectorID)
		require.NoError(t, err)
		require.Equal(t, defaultWorkflowInstances[0], *actual)
	})

	t.Run("unknown connector id", func(t *testing.T) {
		instance := defaultWorkflowInstances[0]
		instance.ConnectorID = models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}

		err := store.InstancesUpsert(ctx, instance)
		require.Error(t, err)
	})

	t.Run("unknown schedule id", func(t *testing.T) {
		instance := defaultWorkflowInstances[0]
		instance.ScheduleID = uuid.New().String()

		err := store.InstancesUpsert(ctx, instance)
		require.Error(t, err)
	})
}

func TestInstancesUpdate(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	for _, schedule := range defaultSchedules {
		upsertSchedule(t, ctx, store, schedule)
	}
	for _, instance := range defaultWorkflowInstances {
		upsertInstance(t, ctx, store, instance)
	}

	t.Run("update instance error", func(t *testing.T) {
		instance := defaultWorkflowInstances[0]
		instance.Error = pointer.For("test error")
		instance.Terminated = true
		instance.TerminatedAt = pointer.For(now.UTC().Time)

		err := store.InstancesUpdate(ctx, instance)
		require.NoError(t, err)

		actual, err := store.InstancesGet(ctx, instance.ID, instance.ScheduleID, instance.ConnectorID)
		require.NoError(t, err)
		require.Equal(t, instance, *actual)
	})

	t.Run("update instance already on error", func(t *testing.T) {
		instance := defaultWorkflowInstances[2]
		instance.Error = pointer.For("test error2")
		instance.Terminated = true
		instance.TerminatedAt = pointer.For(now.UTC().Time)

		err := store.InstancesUpdate(ctx, instance)
		require.NoError(t, err)

		actual, err := store.InstancesGet(ctx, instance.ID, instance.ScheduleID, instance.ConnectorID)
		require.NoError(t, err)
		require.Equal(t, instance, *actual)
	})
}

func TestInstancesDeleteFromConnectorID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	for _, schedule := range defaultSchedules {
		upsertSchedule(t, ctx, store, schedule)
	}
	for _, instance := range defaultWorkflowInstances {
		upsertInstance(t, ctx, store, instance)
	}

	t.Run("delete instances from unknown connector", func(t *testing.T) {
		unknownConnectorID := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}

		require.NoError(t, store.InstancesDeleteFromConnectorID(ctx, unknownConnectorID))

		for _, instance := range defaultWorkflowInstances {
			actual, err := store.InstancesGet(ctx, instance.ID, instance.ScheduleID, instance.ConnectorID)
			require.NoError(t, err)
			require.Equal(t, instance, *actual)
		}
	})

	t.Run("delete instances from default connector", func(t *testing.T) {
		require.NoError(t, store.InstancesDeleteFromConnectorID(ctx, defaultConnector.ID))

		for _, instance := range defaultWorkflowInstances {
			_, err := store.InstancesGet(ctx, instance.ID, instance.ScheduleID, instance.ConnectorID)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrNotFound)
		}
	})
}

func TestInstancesList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	for _, schedule := range defaultSchedules {
		upsertSchedule(t, ctx, store, schedule)
	}
	for _, instance := range defaultWorkflowInstances {
		upsertInstance(t, ctx, store, instance)
	}

	t.Run("wrong query builder operator when listing by schedule id", func(t *testing.T) {
		q := NewListInstancesQuery(
			bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("schedule_id", defaultSchedules[0].ID)),
		)

		cursor, err := store.InstancesList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list instances by schedule_id", func(t *testing.T) {
		q := NewListInstancesQuery(
			bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("schedule_id", defaultSchedules[0].ID)),
		)

		cursor, err := store.InstancesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 2, len(cursor.Data))
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		require.Equal(t, defaultWorkflowInstances[1], cursor.Data[0])
		require.Equal(t, defaultWorkflowInstances[0], cursor.Data[1])
	})

	t.Run("list instances by unknown schedule_id", func(t *testing.T) {
		q := NewListInstancesQuery(
			bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("schedule_id", uuid.New().String())),
		)

		cursor, err := store.InstancesList(ctx, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("list instances by connector_id", func(t *testing.T) {
		q := NewListInstancesQuery(
			bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", defaultConnector.ID)),
		)

		cursor, err := store.InstancesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 3, len(cursor.Data))
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		require.Equal(t, defaultWorkflowInstances[1], cursor.Data[0])
		require.Equal(t, defaultWorkflowInstances[2], cursor.Data[1])
		require.Equal(t, defaultWorkflowInstances[0], cursor.Data[2])
	})

	t.Run("list instances by unknown connector_id", func(t *testing.T) {
		q := NewListInstancesQuery(
			bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "unknown",
				})),
		)

		cursor, err := store.InstancesList(ctx, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("unknown query builder key when listing", func(t *testing.T) {
		q := NewListInstancesQuery(
			bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("unknown", "unknown")),
		)

		cursor, err := store.InstancesList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list instances test cursor", func(t *testing.T) {
		q := NewListInstancesQuery(
			bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).
				WithPageSize(1),
		)

		cursor, err := store.InstancesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 1, len(cursor.Data))
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, defaultWorkflowInstances[1], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.InstancesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 1, len(cursor.Data))
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, defaultWorkflowInstances[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.InstancesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 1, len(cursor.Data))
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		require.Equal(t, defaultWorkflowInstances[0], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.InstancesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 1, len(cursor.Data))
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, defaultWorkflowInstances[2], cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.InstancesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 1, len(cursor.Data))
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, defaultWorkflowInstances[1], cursor.Data[0])
	})
}

func TestInstancesDeleteFromConnectorIDBatch(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	for _, s := range defaultSchedules {
		upsertSchedule(t, ctx, store, s)
	}

	t.Run("invalid batchSize zero", func(t *testing.T) {
		_, err := store.InstancesDeleteFromConnectorIDBatch(ctx, defaultConnector.ID, 0)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidation)
		require.Contains(t, err.Error(), "invalid batchSize 0")
		require.Contains(t, err.Error(), defaultConnector.ID.String())
	})

	t.Run("invalid batchSize negative", func(t *testing.T) {
		_, err := store.InstancesDeleteFromConnectorIDBatch(ctx, defaultConnector.ID, -1)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidation)
		require.Contains(t, err.Error(), "invalid batchSize -1")
		require.Contains(t, err.Error(), defaultConnector.ID.String())
	})

	t.Run("delete batch from unknown connector", func(t *testing.T) {
		rowsAffected, err := store.InstancesDeleteFromConnectorIDBatch(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}, 10)
		require.NoError(t, err)
		require.Equal(t, 0, rowsAffected)
	})

	t.Run("delete batch with no instances", func(t *testing.T) {
		rowsAffected, err := store.InstancesDeleteFromConnectorIDBatch(ctx, defaultConnector.ID, 10)
		require.NoError(t, err)
		require.Equal(t, 0, rowsAffected)
	})

	t.Run("delete single batch smaller than batch size", func(t *testing.T) {
		for _, i := range defaultWorkflowInstances {
			upsertInstance(t, ctx, store, i)
		}

		rowsAffected, err := store.InstancesDeleteFromConnectorIDBatch(ctx, defaultConnector.ID, 10)
		require.NoError(t, err)
		require.Equal(t, 3, rowsAffected)

		// Verify all instances were deleted
		for _, i := range defaultWorkflowInstances {
			_, err := store.InstancesGet(ctx, i.ID, i.ScheduleID, i.ConnectorID)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrNotFound)
		}
	})

	t.Run("delete multiple batches", func(t *testing.T) {
		// Create a connector with more instances
		connector2 := models.Connector{
			ConnectorBase: models.ConnectorBase{
				ID: models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "test2",
				},
				Name:      "test2",
				Provider:  "test2",
				CreatedAt: now.Add(-50 * time.Minute).UTC().Time,
			},
			ScheduledForDeletion: false,
			Config:               []byte(`{}`),
		}
		upsertConnector(t, ctx, store, connector2)

		// Create a schedule for this connector
		schedule2 := models.Schedule{
			ID:          "schedule-test2",
			ConnectorID: connector2.ID,
			CreatedAt:   now.Add(-50 * time.Minute).UTC().Time,
		}
		upsertSchedule(t, ctx, store, schedule2)

		// Create 5 instances for this connector
		instances := make([]models.Instance, 5)
		for i := 0; i < 5; i++ {
			instances[i] = models.Instance{
				ID:          fmt.Sprintf("batch-test-%d", i),
				ScheduleID:  schedule2.ID,
				ConnectorID: connector2.ID,
				CreatedAt:   now.Add(-40 * time.Minute).UTC().Time,
				UpdatedAt:   now.Add(-40 * time.Minute).UTC().Time,
				Terminated:  false,
			}
		}
		for _, i := range instances {
			upsertInstance(t, ctx, store, i)
		}

		// Delete in batches of 2
		totalDeleted := 0
		for {
			rowsAffected, err := store.InstancesDeleteFromConnectorIDBatch(ctx, connector2.ID, 2)
			require.NoError(t, err)
			if rowsAffected == 0 {
				break
			}
			totalDeleted += rowsAffected
		}

		require.Equal(t, 5, totalDeleted)

		// Verify all instances were deleted
		for _, i := range instances {
			_, err := store.InstancesGet(ctx, i.ID, i.ScheduleID, i.ConnectorID)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrNotFound)
		}
	})

	t.Run("delete batch only affects specified connector", func(t *testing.T) {
		// Create two connectors with instances
		connector3 := models.Connector{
			ConnectorBase: models.ConnectorBase{
				ID: models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "test3",
				},
				Name:      "test3",
				Provider:  "test3",
				CreatedAt: now.Add(-45 * time.Minute).UTC().Time,
			},
			ScheduledForDeletion: false,
			Config:               []byte(`{}`),
		}
		upsertConnector(t, ctx, store, connector3)

		connector4 := models.Connector{
			ConnectorBase: models.ConnectorBase{
				ID: models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "test4",
				},
				Name:      "test4",
				Provider:  "test4",
				CreatedAt: now.Add(-45 * time.Minute).UTC().Time,
			},
			ScheduledForDeletion: false,
			Config:               []byte(`{}`),
		}
		upsertConnector(t, ctx, store, connector4)

		// Create schedules for both connectors
		schedule3 := models.Schedule{
			ID:          "schedule-test3",
			ConnectorID: connector3.ID,
			CreatedAt:   now.Add(-45 * time.Minute).UTC().Time,
		}
		upsertSchedule(t, ctx, store, schedule3)

		schedule4 := models.Schedule{
			ID:          "schedule-test4",
			ConnectorID: connector4.ID,
			CreatedAt:   now.Add(-45 * time.Minute).UTC().Time,
		}
		upsertSchedule(t, ctx, store, schedule4)

		// Create 2 instances for connector3
		instances3 := []models.Instance{
			{
				ID:          "isolation-test-1",
				ScheduleID:  schedule3.ID,
				ConnectorID: connector3.ID,
				CreatedAt:   now.Add(-40 * time.Minute).UTC().Time,
				UpdatedAt:   now.Add(-40 * time.Minute).UTC().Time,
				Terminated:  false,
			},
			{
				ID:          "isolation-test-2",
				ScheduleID:  schedule3.ID,
				ConnectorID: connector3.ID,
				CreatedAt:   now.Add(-40 * time.Minute).UTC().Time,
				UpdatedAt:   now.Add(-40 * time.Minute).UTC().Time,
				Terminated:  false,
			},
		}
		for _, i := range instances3 {
			upsertInstance(t, ctx, store, i)
		}

		// Create 1 instance for connector4
		instances4 := []models.Instance{
			{
				ID:          "isolation-test-3",
				ScheduleID:  schedule4.ID,
				ConnectorID: connector4.ID,
				CreatedAt:   now.Add(-40 * time.Minute).UTC().Time,
				UpdatedAt:   now.Add(-40 * time.Minute).UTC().Time,
				Terminated:  false,
			},
		}
		for _, i := range instances4 {
			upsertInstance(t, ctx, store, i)
		}

		// Delete from connector3
		rowsAffected, err := store.InstancesDeleteFromConnectorIDBatch(ctx, connector3.ID, 10)
		require.NoError(t, err)
		require.Equal(t, 2, rowsAffected)

		// Verify connector3 instances are deleted
		for _, i := range instances3 {
			_, err := store.InstancesGet(ctx, i.ID, i.ScheduleID, i.ConnectorID)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrNotFound)
		}

		// Verify connector4 instance still exists
		instance, err := store.InstancesGet(ctx, instances4[0].ID, instances4[0].ScheduleID, instances4[0].ConnectorID)
		require.NoError(t, err)
		require.Equal(t, instances4[0].ID, instance.ID)
	})
}
