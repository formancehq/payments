package storage

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/gibson042/canonicaljson-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	now              = time.Now()
	defaultConnector = models.Connector{
		ID: models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "default",
		},
		Name:      "default",
		CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
		Provider:  "default",
		Config:    []byte(`{}`),
	}

	defaultConnector2 = models.Connector{
		ID: models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "default2",
		},
		Name:      "default2",
		CreatedAt: now.Add(-45 * time.Minute).UTC().Time,
		Provider:  "default2",
		Config:    []byte(`{}`),
	}

	defaultConnector3 = models.Connector{
		ID: models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "default",
		},
		Name:      "default3",
		CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
		Provider:  "default",
		Config:    []byte(`{}`),
	}
)

func upsertConnector(t *testing.T, ctx context.Context, storage Storage, connector models.Connector) {
	require.NoError(t, storage.ConnectorsInstall(ctx, connector))
}

func TestConnectorsInstall(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)

	t.Run("same id upsert", func(t *testing.T) {
		c := models.Connector{
			ID:        defaultConnector.ID,
			Name:      "test changed",
			CreatedAt: time.Now().UTC().Time,
			Provider:  "test",
			Config:    []byte(`{}`),
		}

		require.NoError(t, store.ConnectorsInstall(ctx, c))

		connector, err := store.ConnectorsGet(ctx, c.ID)
		require.NoError(t, err)
		require.NotNil(t, connector)
		require.Equal(t, defaultConnector, *connector)
	})

	t.Run("unique same upsert", func(t *testing.T) {
		c := models.Connector{
			ID: models.ConnectorID{
				Reference: uuid.New(),
				Provider:  "test",
			},
			Name:      "default",
			CreatedAt: now.Add(-23 * time.Minute).UTC().Time,
			Provider:  "test",
			Config:    []byte(`{}`),
		}

		require.Error(t, store.ConnectorsInstall(ctx, c))
	})
}

func TestConnectorsUninstall(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)

	t.Run("uninstall default connector", func(t *testing.T) {
		require.NoError(t, store.ConnectorsUninstall(ctx, defaultConnector.ID))

		connector, err := store.ConnectorsGet(ctx, defaultConnector.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, connector)
	})

	t.Run("uninstall unknown connector", func(t *testing.T) {
		id := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}

		require.NoError(t, store.ConnectorsUninstall(ctx, id))
	})
}

func TestConnectorsConfigUpdate(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)

	t.Run("same id upsert", func(t *testing.T) {
		config := json.RawMessage(`{"val":"new"}`)
		c := models.Connector{
			ID:     defaultConnector.ID,
			Name:   "new name",
			Config: config,
		}

		require.NoError(t, store.ConnectorsConfigUpdate(ctx, c))

		connector, err := store.ConnectorsGet(ctx, c.ID)
		require.NoError(t, err)
		require.NotNil(t, connector)
		assert.Equal(t, c.Name, connector.Name)
		assert.Equal(t, defaultConnector.CreatedAt, connector.CreatedAt)
		assert.Equal(t, defaultConnector.Provider, connector.Provider)
		assert.Equal(t, defaultConnector.ScheduledForDeletion, connector.ScheduledForDeletion)

		expectedData, err := canonicaljson.Marshal(config)
		require.NoError(t, err)
		data, err := canonicaljson.Marshal(connector.Config)
		require.NoError(t, err)
		assert.Equal(t, string(expectedData), string(data))
	})

	t.Run("connector doesn't exist yet", func(t *testing.T) {
		c := models.Connector{
			ID: models.ConnectorID{
				Reference: uuid.New(),
				Provider:  "test",
			},
			Config: []byte(`{}`),
		}

		require.Error(t, store.ConnectorsConfigUpdate(ctx, c))
	})
}

func TestConnectorsScheduleForDeletion(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)

	t.Run("schedule for deletion of unknown connector", func(t *testing.T) {
		id := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}

		require.NoError(t, store.ConnectorsScheduleForDeletion(ctx, id))
	})

	t.Run("schedule for deletion of default connector", func(t *testing.T) {
		require.NoError(t, store.ConnectorsScheduleForDeletion(ctx, defaultConnector.ID))

		connector, err := store.ConnectorsGet(ctx, defaultConnector.ID)
		require.NoError(t, err)
		require.NotNil(t, connector)
		require.True(t, connector.ScheduledForDeletion)
	})
}

func TestConnectorsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)

	t.Run("get connector", func(t *testing.T) {
		connector, err := store.ConnectorsGet(ctx, defaultConnector.ID)
		require.NoError(t, err)
		require.NotNil(t, connector)
		require.Equal(t, defaultConnector, *connector)
	})

	t.Run("get unknown connector", func(t *testing.T) {
		id := models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}

		connector, err := store.ConnectorsGet(ctx, id)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, connector)
	})
}

func TestConnectorsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	upsertConnector(t, ctx, store, defaultConnector3)

	t.Run("list connectors by name", func(t *testing.T) {
		q := NewListConnectorsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConnectorQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("name", "default")),
		)

		cursor, err := store.ConnectorsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Next)
		require.Empty(t, cursor.Previous)
		require.Equal(t, defaultConnector, cursor.Data[0])
	})

	t.Run("list connectors by unknown name", func(t *testing.T) {
		q := NewListConnectorsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConnectorQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("name", "unknown")),
		)

		cursor, err := store.ConnectorsList(ctx, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Next)
		require.Empty(t, cursor.Previous)
	})

	t.Run("list connectors by provider", func(t *testing.T) {
		q := NewListConnectorsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConnectorQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("provider", "default")),
		)

		cursor, err := store.ConnectorsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Next)
		require.Empty(t, cursor.Previous)
		require.Equal(t, defaultConnector3, cursor.Data[0])
		require.Equal(t, defaultConnector, cursor.Data[1])
	})

	t.Run("list connectors by provider uppercase", func(t *testing.T) {
		q := NewListConnectorsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConnectorQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("provider", "DEFAULT")),
		)

		cursor, err := store.ConnectorsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Next)
		require.Empty(t, cursor.Previous)
		require.Equal(t, defaultConnector3, cursor.Data[0])
		require.Equal(t, defaultConnector, cursor.Data[1])
	})

	t.Run("list connectors by provider with wrong type", func(t *testing.T) {
		q := NewListConnectorsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConnectorQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("provider", 1)),
		)

		cursor, err := store.ConnectorsList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list connectors by unknown provider", func(t *testing.T) {
		q := NewListConnectorsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConnectorQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("provider", "unknown")),
		)

		cursor, err := store.ConnectorsList(ctx, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Next)
		require.Empty(t, cursor.Previous)
	})

	t.Run("list connectors by id", func(t *testing.T) {
		q := NewListConnectorsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConnectorQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", defaultConnector3.ID.String())),
		)

		cursor, err := store.ConnectorsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Next)
		require.Empty(t, cursor.Previous)
		require.Equal(t, defaultConnector3, cursor.Data[0])
	})

	t.Run("list connectors by unknown id", func(t *testing.T) {
		q := NewListConnectorsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConnectorQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", "unknown")),
		)

		cursor, err := store.ConnectorsList(ctx, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Next)
		require.Empty(t, cursor.Previous)
	})

	t.Run("list connectors test cursor", func(t *testing.T) {
		q := NewListConnectorsQuery(
			bunpaginate.NewPaginatedQueryOptions(ConnectorQuery{}).
				WithPageSize(1),
		)

		cursor, err := store.ConnectorsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Next)
		require.Empty(t, cursor.Previous)
		require.Equal(t, defaultConnector3, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.ConnectorsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Next)
		require.NotEmpty(t, cursor.Previous)
		require.Equal(t, defaultConnector2, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.ConnectorsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Next)
		require.NotEmpty(t, cursor.Previous)
		require.Equal(t, defaultConnector, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.ConnectorsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Next)
		require.NotEmpty(t, cursor.Previous)
		require.Equal(t, defaultConnector2, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.ConnectorsList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Next)
		require.Empty(t, cursor.Previous)
		require.Equal(t, defaultConnector3, cursor.Data[0])
	})
}
