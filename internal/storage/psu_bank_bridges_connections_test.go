package storage

import (
	"context"
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
	defaultPSUBankBridgeConnectionAttempt = models.PSUBankBridgeConnectionAttempt{
		ID:          uuid.New(),
		PsuID:       defaultPSU2.ID,
		ConnectorID: defaultConnector.ID,
		CreatedAt:   now.Add(-60 * time.Minute).UTC().Time,
		Status:      models.PSUBankBridgeConnectionAttemptStatusPending,
		State: models.CallbackState{
			Randomized: "random123",
			AttemptID:  uuid.New(),
		},
		ClientRedirectURL: pointer.For("https://example.com/redirect"),
		TemporaryToken: &models.Token{
			Token:     "temp_token_123",
			ExpiresAt: now.Add(30 * time.Minute).UTC().Time,
		},
	}

	defaultPSUBankBridgeConnectionAttempt2 = models.PSUBankBridgeConnectionAttempt{
		ID:          uuid.New(),
		PsuID:       defaultPSU2.ID,
		ConnectorID: defaultConnector.ID,
		CreatedAt:   now.Add(-30 * time.Minute).UTC().Time,
		Status:      models.PSUBankBridgeConnectionAttemptStatusCompleted,
		State: models.CallbackState{
			Randomized: "random456",
			AttemptID:  uuid.New(),
		},
	}

	defaultPSUBankBridge = models.PSUBankBridge{
		ConnectorID: defaultConnector.ID,
		AccessToken: &models.Token{
			Token:     "access_token_123",
			ExpiresAt: now.Add(60 * time.Minute).UTC().Time,
		},
		Metadata: map[string]string{
			"foo": "bar",
		},
		Connections: []*models.PSUBankBridgeConnection{
			{
				ConnectorID:   defaultConnector.ID,
				ConnectionID:  "conn_123",
				CreatedAt:     now.Add(-45 * time.Minute).UTC().Time,
				DataUpdatedAt: now.Add(-15 * time.Minute).UTC().Time,
				Status:        models.ConnectionStatusActive,
				AccessToken: &models.Token{
					Token:     "conn_token_123",
					ExpiresAt: now.Add(45 * time.Minute).UTC().Time,
				},
				Metadata: map[string]string{
					"conn_foo": "conn_bar",
				},
			},
		},
	}

	defaultPSUBankBridge2 = models.PSUBankBridge{
		ConnectorID: defaultConnector2.ID,
		AccessToken: &models.Token{
			Token:     "access_token_123",
			ExpiresAt: now.Add(60 * time.Minute).UTC().Time,
		},
	}

	defaultPSUBankBridgeConnection = models.PSUBankBridgeConnection{
		ConnectorID:   defaultConnector.ID,
		ConnectionID:  "conn_456",
		CreatedAt:     now.Add(-40 * time.Minute).UTC().Time,
		DataUpdatedAt: now.Add(-10 * time.Minute).UTC().Time,
		Status:        models.ConnectionStatusActive,
		AccessToken: &models.Token{
			Token:     "conn_token_456",
			ExpiresAt: now.Add(40 * time.Minute).UTC().Time,
		},
		Metadata: map[string]string{
			"conn_foo2": "conn_bar2",
		},
	}

	defaultPSUBankBridgeConnection2 = models.PSUBankBridgeConnection{
		ConnectorID:   defaultConnector.ID,
		ConnectionID:  "conn_789",
		CreatedAt:     now.Add(-35 * time.Minute).UTC().Time,
		DataUpdatedAt: now.Add(-5 * time.Minute).UTC().Time,
		Status:        models.ConnectionStatusError,
		Error:         pointer.For("Connection failed"),
		Metadata: map[string]string{
			"conn_foo3": "conn_bar3",
		},
	}
)

func createPSUBankBridgeConnectionAttempt(t *testing.T, ctx context.Context, storage Storage, attempt models.PSUBankBridgeConnectionAttempt) {
	require.NoError(t, storage.PSUBankBridgeConnectionAttemptsUpsert(ctx, attempt))
}

func createPSUBankBridge(t *testing.T, ctx context.Context, storage Storage, psuID uuid.UUID, bankBridge models.PSUBankBridge) {
	require.NoError(t, storage.PSUBankBridgesUpsert(ctx, psuID, bankBridge))
}

func createPSUBankBridgeConnection(t *testing.T, ctx context.Context, storage Storage, psuID uuid.UUID, connection models.PSUBankBridgeConnection) {
	require.NoError(t, storage.PSUBankBridgeConnectionsUpsert(ctx, psuID, connection))
}

func TestPSUBankBridgeConnectionAttemptsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridgeConnectionAttempt(t, ctx, store, defaultPSUBankBridgeConnectionAttempt)

	t.Run("upsert with same id", func(t *testing.T) {
		attempt := models.PSUBankBridgeConnectionAttempt{
			ID:          defaultPSUBankBridgeConnectionAttempt.ID,
			PsuID:       defaultPSUBankBridgeConnectionAttempt.PsuID,
			ConnectorID: defaultPSUBankBridgeConnectionAttempt.ConnectorID,
			CreatedAt:   now.Add(-50 * time.Minute).UTC().Time,
			Status:      models.PSUBankBridgeConnectionAttemptStatusExited,
			State: models.CallbackState{
				Randomized: "random_changed",
				AttemptID:  uuid.New(),
			},
			ClientRedirectURL: pointer.For("https://example.com/changed"),
			TemporaryToken: &models.Token{
				Token:     "temp_token_changed",
				ExpiresAt: now.Add(20 * time.Minute).UTC().Time,
			},
			Error: pointer.For("Connection failed"),
		}

		require.NoError(t, store.PSUBankBridgeConnectionAttemptsUpsert(ctx, attempt))

		actual, err := store.PSUBankBridgeConnectionAttemptsGet(ctx, defaultPSUBankBridgeConnectionAttempt.ID)
		require.NoError(t, err)
		// Should update the attempt with new values
		require.Equal(t, attempt.Status, actual.Status)
		require.Equal(t, attempt.State, actual.State)
		require.NotNil(t, actual.ClientRedirectURL)
		require.Equal(t, *defaultPSUBankBridgeConnectionAttempt.ClientRedirectURL, *actual.ClientRedirectURL)
		require.Equal(t, attempt.TemporaryToken.Token, actual.TemporaryToken.Token)
		require.Equal(t, attempt.Error, actual.Error)
	})
}

func TestPSUBankBridgeConnectionAttemptsUpdateStatus(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridgeConnectionAttempt(t, ctx, store, defaultPSUBankBridgeConnectionAttempt)

	t.Run("update status to completed", func(t *testing.T) {
		errMsg := pointer.For("Successfully completed")
		require.NoError(t, store.PSUBankBridgeConnectionAttemptsUpdateStatus(ctx, defaultPSUBankBridgeConnectionAttempt.ID, models.PSUBankBridgeConnectionAttemptStatusCompleted, errMsg))

		actual, err := store.PSUBankBridgeConnectionAttemptsGet(ctx, defaultPSUBankBridgeConnectionAttempt.ID)
		require.NoError(t, err)
		require.Equal(t, models.PSUBankBridgeConnectionAttemptStatusCompleted, actual.Status)
		require.Equal(t, errMsg, actual.Error)
	})

	t.Run("update status to failed", func(t *testing.T) {
		errMsg := pointer.For("Connection failed")
		require.NoError(t, store.PSUBankBridgeConnectionAttemptsUpdateStatus(ctx, defaultPSUBankBridgeConnectionAttempt.ID, models.PSUBankBridgeConnectionAttemptStatusExited, errMsg))

		actual, err := store.PSUBankBridgeConnectionAttemptsGet(ctx, defaultPSUBankBridgeConnectionAttempt.ID)
		require.NoError(t, err)
		require.Equal(t, models.PSUBankBridgeConnectionAttemptStatusExited, actual.Status)
		require.Equal(t, errMsg, actual.Error)
	})

	t.Run("update status with nil error", func(t *testing.T) {
		require.NoError(t, store.PSUBankBridgeConnectionAttemptsUpdateStatus(ctx, defaultPSUBankBridgeConnectionAttempt.ID, models.PSUBankBridgeConnectionAttemptStatusPending, nil))

		actual, err := store.PSUBankBridgeConnectionAttemptsGet(ctx, defaultPSUBankBridgeConnectionAttempt.ID)
		require.NoError(t, err)
		require.Equal(t, models.PSUBankBridgeConnectionAttemptStatusPending, actual.Status)
		require.Nil(t, actual.Error)
	})
}

func TestPSUBankBridgeConnectionAttemptsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridgeConnectionAttempt(t, ctx, store, defaultPSUBankBridgeConnectionAttempt)
	createPSUBankBridgeConnectionAttempt(t, ctx, store, defaultPSUBankBridgeConnectionAttempt2)

	t.Run("get attempt with all fields", func(t *testing.T) {
		actual, err := store.PSUBankBridgeConnectionAttemptsGet(ctx, defaultPSUBankBridgeConnectionAttempt.ID)
		require.NoError(t, err)
		comparePSUBankBridgeConnectionAttempts(t, defaultPSUBankBridgeConnectionAttempt, *actual)
	})

	t.Run("get attempt with minimal fields", func(t *testing.T) {
		actual, err := store.PSUBankBridgeConnectionAttemptsGet(ctx, defaultPSUBankBridgeConnectionAttempt2.ID)
		require.NoError(t, err)
		comparePSUBankBridgeConnectionAttempts(t, defaultPSUBankBridgeConnectionAttempt2, *actual)
	})

	t.Run("get non-existent attempt", func(t *testing.T) {
		actual, err := store.PSUBankBridgeConnectionAttemptsGet(ctx, uuid.New())
		require.Error(t, err)
		require.Nil(t, actual)
	})
}

func TestPSUBankBridgeConnectionAttemptsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridgeConnectionAttempt(t, ctx, store, defaultPSUBankBridgeConnectionAttempt)
	createPSUBankBridgeConnectionAttempt(t, ctx, store, defaultPSUBankBridgeConnectionAttempt2)

	t.Run("list attempts by id", func(t *testing.T) {
		q := NewListPSUBankBridgeConnectionAttemptsQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUBankBridgeConnectionAttemptsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", defaultPSUBankBridgeConnectionAttempt.ID.String())),
		)

		cursor, err := store.PSUBankBridgeConnectionAttemptsList(ctx, defaultPSUBankBridgeConnectionAttempt.PsuID, defaultPSUBankBridgeConnectionAttempt.ConnectorID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePSUBankBridgeConnectionAttempts(t, defaultPSUBankBridgeConnectionAttempt, cursor.Data[0])
	})

	t.Run("list attempts by status", func(t *testing.T) {
		q := NewListPSUBankBridgeConnectionAttemptsQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUBankBridgeConnectionAttemptsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("status", string(models.PSUBankBridgeConnectionAttemptStatusCompleted))),
		)

		cursor, err := store.PSUBankBridgeConnectionAttemptsList(ctx, defaultPSUBankBridgeConnectionAttempt2.PsuID, defaultPSUBankBridgeConnectionAttempt2.ConnectorID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePSUBankBridgeConnectionAttempts(t, defaultPSUBankBridgeConnectionAttempt2, cursor.Data[0])
	})

	t.Run("wrong query operator", func(t *testing.T) {
		q := NewListPSUBankBridgeConnectionAttemptsQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUBankBridgeConnectionAttemptsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("id", "test")),
		)

		cursor, err := store.PSUBankBridgeConnectionAttemptsList(ctx, defaultPSUBankBridgeConnectionAttempt.PsuID, defaultPSUBankBridgeConnectionAttempt.ConnectorID, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("unknown query key", func(t *testing.T) {
		q := NewListPSUBankBridgeConnectionAttemptsQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUBankBridgeConnectionAttemptsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("unknown", "test")),
		)

		cursor, err := store.PSUBankBridgeConnectionAttemptsList(ctx, defaultPSUBankBridgeConnectionAttempt.PsuID, defaultPSUBankBridgeConnectionAttempt.ConnectorID, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})
}

func TestPSUBankBridgesUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridge(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridge)

	t.Run("upsert with same psu and connector", func(t *testing.T) {
		bankBridge := models.PSUBankBridge{
			ConnectorID: defaultPSUBankBridge.ConnectorID,
			AccessToken: &models.Token{
				Token:     "access_token_changed",
				ExpiresAt: now.Add(30 * time.Minute).UTC().Time,
			},
			Metadata: map[string]string{
				"changed": "changed",
			},
		}

		require.NoError(t, store.PSUBankBridgesUpsert(ctx, defaultPSU2.ID, bankBridge))

		actual, err := store.PSUBankBridgesGet(ctx, defaultPSU2.ID, defaultPSUBankBridge.ConnectorID)
		require.NoError(t, err)
		// Should update the bank bridge
		require.Equal(t, bankBridge.AccessToken.Token, actual.AccessToken.Token)
		require.Equal(t, bankBridge.Metadata, actual.Metadata)
	})
}

func TestPSUBankBridgesGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridge(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridge)
	createPSUBankBridgeConnection(t, ctx, store, defaultPSU2.ID, *defaultPSUBankBridge.Connections[0])

	t.Run("get bank bridge with connections", func(t *testing.T) {
		actual, err := store.PSUBankBridgesGet(ctx, defaultPSU2.ID, defaultPSUBankBridge.ConnectorID)
		require.NoError(t, err)
		comparePSUBankBridges(t, defaultPSUBankBridge, *actual)
	})

	t.Run("get non-existent bank bridge", func(t *testing.T) {
		actual, err := store.PSUBankBridgesGet(ctx, uuid.New(), defaultPSUBankBridge.ConnectorID)
		require.Error(t, err)
		require.Nil(t, actual)
	})
}

func TestPSUBankBridgesDelete(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridge(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridge)

	t.Run("delete existing bank bridge", func(t *testing.T) {
		require.NoError(t, store.PSUBankBridgesDelete(ctx, defaultPSU2.ID, defaultPSUBankBridge.ConnectorID))

		actual, err := store.PSUBankBridgesGet(ctx, defaultPSU2.ID, defaultPSUBankBridge.ConnectorID)
		require.Error(t, err)
		require.Nil(t, actual)
	})

	t.Run("delete non-existent bank bridge", func(t *testing.T) {
		require.NoError(t, store.PSUBankBridgesDelete(ctx, uuid.New(), defaultPSUBankBridge.ConnectorID))
	})
}

func TestPSUBankBridgesList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridge(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridge)
	createPSUBankBridge(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridge2)

	t.Run("list bank bridges by connector_id", func(t *testing.T) {
		q := NewListPSUBankBridgesQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUBankBridgesQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", defaultPSUBankBridge.ConnectorID.String())),
		)

		cursor, err := store.PSUBankBridgesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
	})

	t.Run("list bank bridges by psu_id", func(t *testing.T) {
		q := NewListPSUBankBridgesQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUBankBridgesQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("psu_id", defaultPSU2.ID.String())),
		)

		cursor, err := store.PSUBankBridgesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
	})

	t.Run("list bank bridges by metadata", func(t *testing.T) {
		q := NewListPSUBankBridgesQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUBankBridgesQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "bar")),
		)

		cursor, err := store.PSUBankBridgesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
	})
}

func TestPSUBankBridgeConnectionsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridgeConnection(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridgeConnection)

	t.Run("upsert with same connection", func(t *testing.T) {
		connection := models.PSUBankBridgeConnection{
			ConnectorID:   defaultPSUBankBridgeConnection.ConnectorID,
			ConnectionID:  defaultPSUBankBridgeConnection.ConnectionID,
			CreatedAt:     now.Add(-35 * time.Minute).UTC().Time,
			DataUpdatedAt: now.Add(-8 * time.Minute).UTC().Time,
			Status:        models.ConnectionStatusError,
			AccessToken: &models.Token{
				Token:     "conn_token_changed",
				ExpiresAt: now.Add(35 * time.Minute).UTC().Time,
			},
			Error: pointer.For("Connection failed"),
			Metadata: map[string]string{
				"changed": "changed",
			},
		}

		require.NoError(t, store.PSUBankBridgeConnectionsUpsert(ctx, defaultPSU2.ID, connection))

		actual, err := store.PSUBankBridgeConnectionsGet(ctx, defaultPSU2.ID, connection.ConnectorID, connection.ConnectionID)
		require.NoError(t, err)
		// Should update the connection
		require.Equal(t, connection.Status, actual.Status)
		require.Equal(t, connection.AccessToken.Token, actual.AccessToken.Token)
		require.Equal(t, connection.Error, actual.Error)
		require.Equal(t, connection.Metadata, actual.Metadata)
	})
}

func TestPSUBankBridgeConnectionsUpdateLastDataUpdate(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridgeConnection(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridgeConnection)

	t.Run("update last data update", func(t *testing.T) {
		newUpdatedAt := now.Add(-5 * time.Minute).UTC().Time
		require.NoError(t, store.PSUBankBridgeConnectionsUpdateLastDataUpdate(ctx, defaultPSU2.ID, defaultPSUBankBridgeConnection.ConnectorID, defaultPSUBankBridgeConnection.ConnectionID, newUpdatedAt))

		actual, err := store.PSUBankBridgeConnectionsGet(ctx, defaultPSU2.ID, defaultPSUBankBridgeConnection.ConnectorID, defaultPSUBankBridgeConnection.ConnectionID)
		require.NoError(t, err)
		require.Equal(t, newUpdatedAt, actual.DataUpdatedAt)
	})
}

func TestPSUBankBridgeConnectionsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridgeConnection(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridgeConnection)
	createPSUBankBridgeConnection(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridgeConnection2)

	t.Run("get connection with all fields", func(t *testing.T) {
		actual, err := store.PSUBankBridgeConnectionsGet(ctx, defaultPSU2.ID, defaultPSUBankBridgeConnection.ConnectorID, defaultPSUBankBridgeConnection.ConnectionID)
		require.NoError(t, err)
		comparePSUBankBridgeConnections(t, defaultPSUBankBridgeConnection, *actual)
	})

	t.Run("get connection with error", func(t *testing.T) {
		actual, err := store.PSUBankBridgeConnectionsGet(ctx, defaultPSU2.ID, defaultPSUBankBridgeConnection2.ConnectorID, defaultPSUBankBridgeConnection2.ConnectionID)
		require.NoError(t, err)
		comparePSUBankBridgeConnections(t, defaultPSUBankBridgeConnection2, *actual)
	})

	t.Run("get non-existent connection", func(t *testing.T) {
		actual, err := store.PSUBankBridgeConnectionsGet(ctx, defaultPSU2.ID, defaultPSUBankBridgeConnection.ConnectorID, "non_existent")
		require.Error(t, err)
		require.Nil(t, actual)
	})
}

func TestPSUBankBridgeConnectionsDelete(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridgeConnection(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridgeConnection)

	t.Run("delete existing connection", func(t *testing.T) {
		require.NoError(t, store.PSUBankBridgeConnectionsDelete(ctx, defaultPSU2.ID, defaultPSUBankBridgeConnection.ConnectorID, defaultPSUBankBridgeConnection.ConnectionID))

		actual, err := store.PSUBankBridgeConnectionsGet(ctx, defaultPSU2.ID, defaultPSUBankBridgeConnection.ConnectorID, defaultPSUBankBridgeConnection.ConnectionID)
		require.Error(t, err)
		require.Nil(t, actual)
	})

	t.Run("delete non-existent connection", func(t *testing.T) {
		require.NoError(t, store.PSUBankBridgeConnectionsDelete(ctx, defaultPSU2.ID, defaultPSUBankBridgeConnection.ConnectorID, "non_existent"))
	})
}

func TestPSUBankBridgeConnectionsGetFromConnectionID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridgeConnection(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridgeConnection)

	t.Run("get connection by connection ID", func(t *testing.T) {
		actual, actualPsuID, err := store.PSUBankBridgeConnectionsGetFromConnectionID(ctx, defaultPSUBankBridgeConnection.ConnectorID, defaultPSUBankBridgeConnection.ConnectionID)
		require.NoError(t, err)
		require.Equal(t, defaultPSU2.ID, actualPsuID)
		comparePSUBankBridgeConnections(t, defaultPSUBankBridgeConnection, *actual)
	})

	t.Run("get non-existent connection by connection ID", func(t *testing.T) {
		actual, actualPsuID, err := store.PSUBankBridgeConnectionsGetFromConnectionID(ctx, defaultPSUBankBridgeConnection.ConnectorID, "non_existent")
		require.Error(t, err)
		require.Equal(t, uuid.Nil, actualPsuID)
		require.Nil(t, actual)
	})
}

func TestPSUBankBridgeConnectionsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridgeConnection(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridgeConnection)
	createPSUBankBridgeConnection(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridgeConnection2)

	t.Run("list connections by connection_id", func(t *testing.T) {
		q := NewListPsuBankBridgeConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuBankBridgeConnectionsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connection_id", defaultPSUBankBridgeConnection.ConnectionID)),
		)

		cursor, err := store.PSUBankBridgeConnectionsList(ctx, defaultPSU2.ID, nil, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePSUBankBridgeConnections(t, defaultPSUBankBridgeConnection, cursor.Data[0])
	})

	t.Run("list connections by status", func(t *testing.T) {
		q := NewListPsuBankBridgeConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuBankBridgeConnectionsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("status", string(models.ConnectionStatusError))),
		)

		cursor, err := store.PSUBankBridgeConnectionsList(ctx, defaultPSU2.ID, nil, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePSUBankBridgeConnections(t, defaultPSUBankBridgeConnection2, cursor.Data[0])
	})

	t.Run("list connections by metadata", func(t *testing.T) {
		q := NewListPsuBankBridgeConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuBankBridgeConnectionsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[conn_foo2]", "conn_bar2")),
		)

		cursor, err := store.PSUBankBridgeConnectionsList(ctx, defaultPSU2.ID, nil, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePSUBankBridgeConnections(t, defaultPSUBankBridgeConnection, cursor.Data[0])
	})

	t.Run("list connections with connector filter", func(t *testing.T) {
		q := NewListPsuBankBridgeConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuBankBridgeConnectionsQuery{}).
				WithPageSize(15),
		)

		cursor, err := store.PSUBankBridgeConnectionsList(ctx, defaultPSU2.ID, &defaultPSUBankBridgeConnection.ConnectorID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
	})

	t.Run("wrong query operator", func(t *testing.T) {
		q := NewListPsuBankBridgeConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuBankBridgeConnectionsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("connection_id", "test")),
		)

		cursor, err := store.PSUBankBridgeConnectionsList(ctx, defaultPSU2.ID, nil, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("unknown query key", func(t *testing.T) {
		q := NewListPsuBankBridgeConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuBankBridgeConnectionsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("unknown", "test")),
		)

		cursor, err := store.PSUBankBridgeConnectionsList(ctx, defaultPSU2.ID, nil, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})
}

func comparePSUBankBridgeConnectionAttempts(t *testing.T, expected, actual models.PSUBankBridgeConnectionAttempt) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.PsuID, actual.PsuID)
	require.Equal(t, expected.ConnectorID, actual.ConnectorID)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.State, actual.State)
	require.Equal(t, expected.ClientRedirectURL, actual.ClientRedirectURL)
	require.Equal(t, expected.Error, actual.Error)

	if expected.TemporaryToken != nil {
		require.NotNil(t, actual.TemporaryToken)
		require.Equal(t, expected.TemporaryToken.Token, actual.TemporaryToken.Token)
		require.Equal(t, expected.TemporaryToken.ExpiresAt, actual.TemporaryToken.ExpiresAt)
	} else {
		require.Nil(t, actual.TemporaryToken)
	}
}

func comparePSUBankBridges(t *testing.T, expected, actual models.PSUBankBridge) {
	require.Equal(t, expected.ConnectorID, actual.ConnectorID)
	require.Equal(t, expected.Metadata, actual.Metadata)

	if expected.AccessToken != nil {
		require.NotNil(t, actual.AccessToken)
		require.Equal(t, expected.AccessToken.Token, actual.AccessToken.Token)
		require.Equal(t, expected.AccessToken.ExpiresAt, actual.AccessToken.ExpiresAt)
	} else {
		require.Nil(t, actual.AccessToken)
	}

	require.Len(t, actual.Connections, len(expected.Connections))
	for i, expectedConnection := range expected.Connections {
		comparePSUBankBridgeConnections(t, *expectedConnection, *actual.Connections[i])
	}
}

func comparePSUBankBridgeConnections(t *testing.T, expected, actual models.PSUBankBridgeConnection) {
	require.Equal(t, expected.ConnectorID, actual.ConnectorID)
	require.Equal(t, expected.ConnectionID, actual.ConnectionID)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.DataUpdatedAt, actual.DataUpdatedAt)
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.Error, actual.Error)
	require.Equal(t, expected.Metadata, actual.Metadata)

	if expected.AccessToken != nil {
		require.NotNil(t, actual.AccessToken)
		require.Equal(t, expected.AccessToken.Token, actual.AccessToken.Token)
		require.Equal(t, expected.AccessToken.ExpiresAt, actual.AccessToken.ExpiresAt)
	} else {
		require.Nil(t, actual.AccessToken)
	}
}
