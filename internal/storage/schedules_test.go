package storage

import (
	"context"
	"github.com/formancehq/go-libs/v3/platform/postgres"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	defaultSchedules = []models.Schedule{
		{
			ID:          "test1",
			ConnectorID: defaultConnector.ID,
			CreatedAt:   now.Add(-60 * time.Minute).UTC().Time,
		},
		{
			ID:          "test2",
			ConnectorID: defaultConnector.ID,
			CreatedAt:   now.Add(-30 * time.Minute).UTC().Time,
		},
		{
			ID:          "test3",
			ConnectorID: defaultConnector.ID,
			CreatedAt:   now.Add(-55 * time.Minute).UTC().Time,
		},
	}
)

func upsertSchedule(t *testing.T, ctx context.Context, storage Storage, schedule models.Schedule) {
	require.NoError(t, storage.SchedulesUpsert(ctx, schedule))
}

func TestSchedulesUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertSchedule(t, ctx, store, defaultSchedules[0])
	upsertSchedule(t, ctx, store, defaultSchedules[1])
	upsertSchedule(t, ctx, store, defaultSchedules[2])

	t.Run("upsert with same id", func(t *testing.T) {
		sch := models.Schedule{
			ID:          "test1",
			ConnectorID: defaultConnector.ID,
			CreatedAt:   now.Add(-90 * time.Minute).UTC().Time,
		}

		require.NoError(t, store.SchedulesUpsert(ctx, sch))

		actual, err := store.SchedulesGet(ctx, sch.ID, sch.ConnectorID)
		require.NoError(t, err)
		require.Equal(t, defaultSchedules[0], *actual)
	})

	t.Run("upsert with unknown connector id", func(t *testing.T) {
		sch := models.Schedule{
			ID: "test4",
			ConnectorID: models.ConnectorID{
				Reference: uuid.New(),
				Provider:  "unknown",
			},
			CreatedAt: now.Add(-90 * time.Minute).UTC().Time,
		}

		require.Error(t, store.SchedulesUpsert(ctx, sch))
	})
}

func TestSchedulesDeleteFromConnectorID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertSchedule(t, ctx, store, defaultSchedules[0])
	upsertSchedule(t, ctx, store, defaultSchedules[1])
	upsertSchedule(t, ctx, store, defaultSchedules[2])

	t.Run("delete schedules from unknown connector id", func(t *testing.T) {
		require.NoError(t, store.SchedulesDeleteFromConnectorID(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}))

		for _, sch := range defaultSchedules {
			actual, err := store.SchedulesGet(ctx, sch.ID, sch.ConnectorID)
			require.NoError(t, err)
			require.Equal(t, sch, *actual)
		}
	})

	t.Run("delete schedules", func(t *testing.T) {
		require.NoError(t, store.SchedulesDeleteFromConnectorID(ctx, defaultConnector.ID))

		for _, sch := range defaultSchedules {
			_, err := store.SchedulesGet(ctx, sch.ID, sch.ConnectorID)
			require.Error(t, err)
			require.ErrorIs(t, err, postgres.ErrNotFound)
		}
	})
}

func TestSchedulesDelete(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertSchedule(t, ctx, store, defaultSchedules[0])
	upsertSchedule(t, ctx, store, defaultSchedules[1])
	upsertSchedule(t, ctx, store, defaultSchedules[2])

	t.Run("delete unknown schedule", func(t *testing.T) {
		require.NoError(t, store.SchedulesDelete(ctx, "unknown"))

		for _, sch := range defaultSchedules {
			actual, err := store.SchedulesGet(ctx, sch.ID, sch.ConnectorID)
			require.NoError(t, err)
			require.Equal(t, sch, *actual)
		}
	})

	t.Run("delete schedule", func(t *testing.T) {
		require.NoError(t, store.SchedulesDelete(ctx, defaultSchedules[0].ID))

		_, err := store.SchedulesGet(ctx, defaultSchedules[0].ID, defaultSchedules[0].ConnectorID)
		require.Error(t, err)
		require.ErrorIs(t, err, postgres.ErrNotFound)
	})
}

func TestSchedulesGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertSchedule(t, ctx, store, defaultSchedules[0])
	upsertSchedule(t, ctx, store, defaultSchedules[1])
	upsertSchedule(t, ctx, store, defaultSchedules[2])

	t.Run("get schedule", func(t *testing.T) {
		actual, err := store.SchedulesGet(ctx, defaultSchedules[0].ID, defaultSchedules[0].ConnectorID)
		require.NoError(t, err)
		require.Equal(t, defaultSchedules[0], *actual)
	})

	t.Run("get unknown schedule", func(t *testing.T) {
		_, err := store.SchedulesGet(ctx, "unknown", defaultConnector.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, postgres.ErrNotFound)
	})
}

func TestSchedulesList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertSchedule(t, ctx, store, defaultSchedules[0])
	upsertSchedule(t, ctx, store, defaultSchedules[1])
	upsertSchedule(t, ctx, store, defaultSchedules[2])

	t.Run("wrong query builder operator when listing by connector_id", func(t *testing.T) {
		q := NewListSchedulesQuery(
			bunpaginate.NewPaginatedQueryOptions(ScheduleQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("connector_id", defaultConnector.ID)),
		)

		cursor, err := store.SchedulesList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
		assert.True(t, errors.Is(err, ErrValidation))
		assert.Regexp(t, "connector_id", err.Error())
	})

	t.Run("list schedules by connector id", func(t *testing.T) {
		q := NewListSchedulesQuery(
			bunpaginate.NewPaginatedQueryOptions(ScheduleQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", defaultConnector.ID)),
		)

		cursor, err := store.SchedulesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 15, cursor.PageSize)
		require.Equal(t, 3, len(cursor.Data))
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		require.Equal(t, []models.Schedule{defaultSchedules[1], defaultSchedules[2], defaultSchedules[0]}, cursor.Data)
	})

	t.Run("list schedules by unknown connector id", func(t *testing.T) {
		q := NewListSchedulesQuery(
			bunpaginate.NewPaginatedQueryOptions(ScheduleQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "unknown",
				}),
				),
		)

		cursor, err := store.SchedulesList(ctx, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("list schedules by id", func(t *testing.T) {
		q := NewListSchedulesQuery(
			bunpaginate.NewPaginatedQueryOptions(ScheduleQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", defaultSchedules[1].ID)),
		)

		cursor, err := store.SchedulesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 15, cursor.PageSize)
		require.Equal(t, 1, len(cursor.Data))
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		require.Equal(t, []models.Schedule{defaultSchedules[1]}, cursor.Data)
	})

	t.Run("list schedules by unknown id", func(t *testing.T) {
		q := NewListSchedulesQuery(
			bunpaginate.NewPaginatedQueryOptions(ScheduleQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "unknown",
				}),
				),
		)

		cursor, err := store.SchedulesList(ctx, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("unknown query builder key when listing", func(t *testing.T) {
		q := NewListSchedulesQuery(
			bunpaginate.NewPaginatedQueryOptions(ScheduleQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("unknown", "unknown")),
		)

		cursor, err := store.SchedulesList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list schedules test cursor", func(t *testing.T) {
		q := NewListSchedulesQuery(
			bunpaginate.NewPaginatedQueryOptions(ScheduleQuery{}).
				WithPageSize(1),
		)

		cursor, err := store.SchedulesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 1, len(cursor.Data))
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, []models.Schedule{defaultSchedules[1]}, cursor.Data)

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.SchedulesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 1, len(cursor.Data))
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, []models.Schedule{defaultSchedules[2]}, cursor.Data)

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.SchedulesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 1, len(cursor.Data))
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		require.Equal(t, []models.Schedule{defaultSchedules[0]}, cursor.Data)

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.SchedulesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 1, len(cursor.Data))
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, []models.Schedule{defaultSchedules[2]}, cursor.Data)

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.SchedulesList(ctx, q)
		require.NoError(t, err)
		require.Equal(t, 1, len(cursor.Data))
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		require.Equal(t, []models.Schedule{defaultSchedules[1]}, cursor.Data)
	})
}
