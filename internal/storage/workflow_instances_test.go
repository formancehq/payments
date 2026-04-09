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


func TestInstancesListSchedulesAboveErrorThreshold(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	for _, schedule := range defaultSchedules {
		upsertSchedule(t, ctx, store, schedule)
	}

	// scheduleID → schedule index mapping for readability
	// defaultSchedules[0].ID = "test1"
	// defaultSchedules[1].ID = "test2"
	// defaultSchedules[2].ID = "test3"

	makeInstance := func(id string, scheduleIdx int, offsetMinutes int, err *string) models.Instance {
		t.Helper()
		return models.Instance{
			ID:          id,
			ScheduleID:  defaultSchedules[scheduleIdx].ID,
			ConnectorID: defaultConnector.ID,
			CreatedAt:   now.Add(time.Duration(-offsetMinutes) * time.Minute).UTC().Time,
			UpdatedAt:   now.Add(time.Duration(-offsetMinutes) * time.Minute).UTC().Time,
			Terminated:  err != nil,
			Error:       err,
		}
	}

	t.Run("returns schedule when last 5 executions all have errors", func(t *testing.T) {
		store := newStore(t)
		defer store.Close()
		upsertConnector(t, ctx, store, defaultConnector)
		for _, s := range defaultSchedules {
			upsertSchedule(t, ctx, store, s)
		}

		for i := 0; i < 5; i++ {
			upsertInstance(t, ctx, store, makeInstance(fmt.Sprintf("all-err-%d", i), 0, 10+i, pointer.For("error")))
		}

		q := NewListInstancesQuery(bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).WithPageSize(15))
		cursor, err := store.InstancesListSchedulesAboveErrorThreshold(ctx, defaultConnector.ID, 5, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.Equal(t, defaultSchedules[0].ID, cursor.Data[0].ScheduleID)
	})

	t.Run("excludes schedule when any of last 5 executions has no error", func(t *testing.T) {
		store := newStore(t)
		defer store.Close()
		upsertConnector(t, ctx, store, defaultConnector)
		for _, s := range defaultSchedules {
			upsertSchedule(t, ctx, store, s)
		}

		// 4 errors then 1 success (most recent is success = offset 10)
		upsertInstance(t, ctx, store, makeInstance("mixed-ok", 0, 10, nil))
		for i := 1; i <= 4; i++ {
			upsertInstance(t, ctx, store, makeInstance(fmt.Sprintf("mixed-err-%d", i), 0, 10+i, pointer.For("error")))
		}

		q := NewListInstancesQuery(bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).WithPageSize(15))
		cursor, err := store.InstancesListSchedulesAboveErrorThreshold(ctx, defaultConnector.ID, 5, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
	})

	t.Run("excludes schedule with fewer than 5 executions", func(t *testing.T) {
		store := newStore(t)
		defer store.Close()
		upsertConnector(t, ctx, store, defaultConnector)
		for _, s := range defaultSchedules {
			upsertSchedule(t, ctx, store, s)
		}

		for i := 0; i < 3; i++ {
			upsertInstance(t, ctx, store, makeInstance(fmt.Sprintf("few-err-%d", i), 0, 10+i, pointer.For("error")))
		}

		q := NewListInstancesQuery(bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).WithPageSize(15))
		cursor, err := store.InstancesListSchedulesAboveErrorThreshold(ctx, defaultConnector.ID, 5, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
	})

	t.Run("ignores executions beyond the 5 most recent", func(t *testing.T) {
		store := newStore(t)
		defer store.Close()
		upsertConnector(t, ctx, store, defaultConnector)
		for _, s := range defaultSchedules {
			upsertSchedule(t, ctx, store, s)
		}

		// 5 most recent are errors, but an older (6th) one had no error
		upsertInstance(t, ctx, store, makeInstance("old-ok", 0, 100, nil))
		for i := 0; i < 5; i++ {
			upsertInstance(t, ctx, store, makeInstance(fmt.Sprintf("recent-err-%d", i), 0, 10+i, pointer.For("error")))
		}

		q := NewListInstancesQuery(bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).WithPageSize(15))
		cursor, err := store.InstancesListSchedulesAboveErrorThreshold(ctx, defaultConnector.ID, 5, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.Equal(t, defaultSchedules[0].ID, cursor.Data[0].ScheduleID)
	})

	t.Run("returns most recent error instance per schedule", func(t *testing.T) {
		store := newStore(t)
		defer store.Close()
		upsertConnector(t, ctx, store, defaultConnector)
		for _, s := range defaultSchedules {
			upsertSchedule(t, ctx, store, s)
		}

		// schedule[0]: 5 errors, most recent has error "latest"
		upsertInstance(t, ctx, store, makeInstance("sched0-oldest", 0, 50, pointer.For("old error")))
		for i := 1; i <= 4; i++ {
			upsertInstance(t, ctx, store, makeInstance(fmt.Sprintf("sched0-%d", i), 0, 10+i, pointer.For("error")))
		}
		upsertInstance(t, ctx, store, makeInstance("sched0-latest", 0, 5, pointer.For("latest")))

		q := NewListInstancesQuery(bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).WithPageSize(15))
		cursor, err := store.InstancesListSchedulesAboveErrorThreshold(ctx, defaultConnector.ID, 5, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.Equal(t, pointer.For("latest"), cursor.Data[0].Error)
	})

	t.Run("returns multiple qualifying schedules", func(t *testing.T) {
		store := newStore(t)
		defer store.Close()
		upsertConnector(t, ctx, store, defaultConnector)
		for _, s := range defaultSchedules {
			upsertSchedule(t, ctx, store, s)
		}

		// schedule[0]: 5 errors
		for i := 0; i < 5; i++ {
			upsertInstance(t, ctx, store, makeInstance(fmt.Sprintf("s0-%d", i), 0, 10+i, pointer.For("err")))
		}
		// schedule[1]: 5 errors
		for i := 0; i < 5; i++ {
			upsertInstance(t, ctx, store, makeInstance(fmt.Sprintf("s1-%d", i), 1, 10+i, pointer.For("err")))
		}
		// schedule[2]: has a success in the last 5 → excluded
		upsertInstance(t, ctx, store, makeInstance("s2-ok", 2, 10, nil))
		for i := 1; i < 5; i++ {
			upsertInstance(t, ctx, store, makeInstance(fmt.Sprintf("s2-err-%d", i), 2, 10+i, pointer.For("err")))
		}

		q := NewListInstancesQuery(bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).WithPageSize(15))
		cursor, err := store.InstancesListSchedulesAboveErrorThreshold(ctx, defaultConnector.ID, 5, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		scheduleIDs := []string{cursor.Data[0].ScheduleID, cursor.Data[1].ScheduleID}
		require.Contains(t, scheduleIDs, defaultSchedules[0].ID)
		require.Contains(t, scheduleIDs, defaultSchedules[1].ID)
	})

	t.Run("cursor pagination returns next page", func(t *testing.T) {
		store := newStore(t)
		defer store.Close()
		upsertConnector(t, ctx, store, defaultConnector)
		for _, s := range defaultSchedules {
			upsertSchedule(t, ctx, store, s)
		}

		// 3 schedules each with 5 errors
		for sIdx := 0; sIdx < 3; sIdx++ {
			for i := 0; i < 5; i++ {
				upsertInstance(t, ctx, store, makeInstance(fmt.Sprintf("page-s%d-%d", sIdx, i), sIdx, 10+i, pointer.For("err")))
			}
		}

		q := NewListInstancesQuery(bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).WithPageSize(1))
		page1, err := store.InstancesListSchedulesAboveErrorThreshold(ctx, defaultConnector.ID, 5, q)
		require.NoError(t, err)
		require.Len(t, page1.Data, 1)
		require.True(t, page1.HasMore)
		require.NotEmpty(t, page1.Next)

		err = bunpaginate.UnmarshalCursor(page1.Next, &q)
		require.NoError(t, err)
		page2, err := store.InstancesListSchedulesAboveErrorThreshold(ctx, defaultConnector.ID, 5, q)
		require.NoError(t, err)
		require.Len(t, page2.Data, 1)
		require.True(t, page2.HasMore)
		require.NotEqual(t, page1.Data[0].ScheduleID, page2.Data[0].ScheduleID)

		err = bunpaginate.UnmarshalCursor(page2.Next, &q)
		require.NoError(t, err)
		page3, err := store.InstancesListSchedulesAboveErrorThreshold(ctx, defaultConnector.ID, 5, q)
		require.NoError(t, err)
		require.Len(t, page3.Data, 1)
		require.False(t, page3.HasMore)
		require.NotEqual(t, page2.Data[0].ScheduleID, page3.Data[0].ScheduleID)
	})

	t.Run("excludes non-terminated instances even when they have errors", func(t *testing.T) {
		store := newStore(t)
		defer store.Close()
		upsertConnector(t, ctx, store, defaultConnector)
		for _, s := range defaultSchedules {
			upsertSchedule(t, ctx, store, s)
		}

		// Insert 5 instances with errors but terminated=false (still running).
		for i := 0; i < 5; i++ {
			inst := makeInstance(fmt.Sprintf("running-err-%d", i), 0, 10+i, pointer.For("error"))
			inst.Terminated = false
			upsertInstance(t, ctx, store, inst)
		}

		q := NewListInstancesQuery(bunpaginate.NewPaginatedQueryOptions(InstanceQuery{}).WithPageSize(15))
		cursor, err := store.InstancesListSchedulesAboveErrorThreshold(ctx, defaultConnector.ID, 5, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
	})

	_ = store // used for setup above
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
