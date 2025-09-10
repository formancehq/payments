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
	defaultOpenBankingConnectionAttempt = models.OpenBankingConnectionAttempt{
		ID:          uuid.New(),
		PsuID:       defaultPSU2.ID,
		ConnectorID: defaultConnector.ID,
		CreatedAt:   now.Add(-60 * time.Minute).UTC().Time,
		Status:      models.OpenBankingConnectionAttemptStatusPending,
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

	defaultOpenBankingConnectionAttempt2 = models.OpenBankingConnectionAttempt{
		ID:          uuid.New(),
		PsuID:       defaultPSU2.ID,
		ConnectorID: defaultConnector.ID,
		CreatedAt:   now.Add(-30 * time.Minute).UTC().Time,
		Status:      models.OpenBankingConnectionAttemptStatusCompleted,
		State: models.CallbackState{
			Randomized: "random456",
			AttemptID:  uuid.New(),
		},
	}

	defaultPSUBankBridge = models.OpenBankingProviderPSU{
		ConnectorID: defaultConnector.ID,
		AccessToken: &models.Token{
			Token:     "access_token_123",
			ExpiresAt: now.Add(60 * time.Minute).UTC().Time,
		},
		Metadata: map[string]string{
			"foo": "bar",
		},
	}

	bankBridgeConn = models.OpenBankingConnection{
		ConnectorID:   defaultConnector.ID,
		ConnectionID:  "conn_123",
		CreatedAt:     now.Add(-45 * time.Minute).UTC().Time,
		DataUpdatedAt: now.Add(-15 * time.Minute).UTC().Time,
		Status:        models.ConnectionStatusActive,
		UpdatedAt:     now.Add(-15 * time.Minute).UTC().Time,
		AccessToken: &models.Token{
			Token:     "conn_token_123",
			ExpiresAt: now.Add(45 * time.Minute).UTC().Time,
		},
		Metadata: map[string]string{
			"conn_foo": "conn_bar",
		},
	}

	defaultPSUBankBridge2 = models.OpenBankingProviderPSU{
		ConnectorID: defaultConnector2.ID,
		AccessToken: &models.Token{
			Token:     "access_token_123",
			ExpiresAt: now.Add(60 * time.Minute).UTC().Time,
		},
	}

	defaultPSUBankBridgeWithPSPUserID = models.OpenBankingProviderPSU{
		ConnectorID: defaultConnector.ID,
		PSPUserID:   pointer.For("psp_user_123"),
		AccessToken: &models.Token{
			Token:     "access_token_psp_123",
			ExpiresAt: now.Add(60 * time.Minute).UTC().Time,
		},
		Metadata: map[string]string{
			"psp_foo": "psp_bar",
		},
	}

	defaultPSUBankBridgeWithPSPUserID2 = models.OpenBankingProviderPSU{
		ConnectorID: defaultConnector2.ID,
		PSPUserID:   pointer.For("psp_user_456"),
		AccessToken: &models.Token{
			Token:     "access_token_psp_456",
			ExpiresAt: now.Add(60 * time.Minute).UTC().Time,
		},
		Metadata: map[string]string{
			"psp_foo2": "psp_bar2",
		},
	}

	defaultOpenBankingConnection = models.OpenBankingConnection{
		ConnectorID:   defaultConnector.ID,
		ConnectionID:  "conn_456",
		CreatedAt:     now.Add(-40 * time.Minute).UTC().Time,
		DataUpdatedAt: now.Add(-10 * time.Minute).UTC().Time,
		Status:        models.ConnectionStatusActive,
		UpdatedAt:     now.Add(-10 * time.Minute).UTC().Time,
		AccessToken: &models.Token{
			Token:     "conn_token_456",
			ExpiresAt: now.Add(40 * time.Minute).UTC().Time,
		},
		Metadata: map[string]string{
			"conn_foo2": "conn_bar2",
		},
	}

	defaultOpenBankingConnection2 = models.OpenBankingConnection{
		ConnectorID:   defaultConnector.ID,
		ConnectionID:  "conn_789",
		CreatedAt:     now.Add(-35 * time.Minute).UTC().Time,
		DataUpdatedAt: now.Add(-5 * time.Minute).UTC().Time,
		UpdatedAt:     now.Add(-5 * time.Minute).UTC().Time,
		Status:        models.ConnectionStatusError,
		Error:         pointer.For("Connection failed"),
		Metadata: map[string]string{
			"conn_foo3": "conn_bar3",
		},
	}
)

func createOpenBankingConnectionAttempt(t *testing.T, ctx context.Context, storage Storage, attempt models.OpenBankingConnectionAttempt) {
	require.NoError(t, storage.OpenBankingConnectionAttemptsUpsert(ctx, attempt))
}

func createPSUBankBridge(t *testing.T, ctx context.Context, storage Storage, psuID uuid.UUID, forwardedUser models.OpenBankingProviderPSU) {
	require.NoError(t, storage.OpenBankingProviderPSUUpsert(ctx, psuID, forwardedUser))
}

func createOpenBankingConnection(t *testing.T, ctx context.Context, storage Storage, psuID uuid.UUID, connection models.OpenBankingConnection) {
	require.NoError(t, storage.PSUOpenBankingConnectionsUpsert(ctx, psuID, connection))
}

func TestOpenBankingConnectionAttemptsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnectionAttempt(t, ctx, store, defaultOpenBankingConnectionAttempt)

	t.Run("upsert with same id", func(t *testing.T) {
		attempt := models.OpenBankingConnectionAttempt{
			ID:          defaultOpenBankingConnectionAttempt.ID,
			PsuID:       defaultOpenBankingConnectionAttempt.PsuID,
			ConnectorID: defaultOpenBankingConnectionAttempt.ConnectorID,
			CreatedAt:   now.Add(-50 * time.Minute).UTC().Time,
			Status:      models.OpenBankingConnectionAttemptStatusExited,
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

		require.NoError(t, store.OpenBankingConnectionAttemptsUpsert(ctx, attempt))

		actual, err := store.OpenBankingConnectionAttemptsGet(ctx, defaultOpenBankingConnectionAttempt.ID)
		require.NoError(t, err)
		// Should update the attempt with new values
		require.Equal(t, attempt.Status, actual.Status)
		require.Equal(t, attempt.State, actual.State)
		require.NotNil(t, actual.ClientRedirectURL)
		require.Equal(t, *defaultOpenBankingConnectionAttempt.ClientRedirectURL, *actual.ClientRedirectURL)
		require.Equal(t, attempt.TemporaryToken.Token, actual.TemporaryToken.Token)
		require.Equal(t, attempt.Error, actual.Error)
	})
}

func TestOpenBankingConnectionAttemptsUpdateStatus(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnectionAttempt(t, ctx, store, defaultOpenBankingConnectionAttempt)

	t.Run("update status to completed", func(t *testing.T) {
		errMsg := pointer.For("Successfully completed")
		require.NoError(t, store.OpenBankingConnectionAttemptsUpdateStatus(ctx, defaultOpenBankingConnectionAttempt.ID, models.OpenBankingConnectionAttemptStatusCompleted, errMsg))

		actual, err := store.OpenBankingConnectionAttemptsGet(ctx, defaultOpenBankingConnectionAttempt.ID)
		require.NoError(t, err)
		require.Equal(t, models.OpenBankingConnectionAttemptStatusCompleted, actual.Status)
		require.Equal(t, errMsg, actual.Error)
	})

	t.Run("update status to failed", func(t *testing.T) {
		errMsg := pointer.For("Connection failed")
		require.NoError(t, store.OpenBankingConnectionAttemptsUpdateStatus(ctx, defaultOpenBankingConnectionAttempt.ID, models.OpenBankingConnectionAttemptStatusExited, errMsg))

		actual, err := store.OpenBankingConnectionAttemptsGet(ctx, defaultOpenBankingConnectionAttempt.ID)
		require.NoError(t, err)
		require.Equal(t, models.OpenBankingConnectionAttemptStatusExited, actual.Status)
		require.Equal(t, errMsg, actual.Error)
	})

	t.Run("update status with nil error", func(t *testing.T) {
		require.NoError(t, store.OpenBankingConnectionAttemptsUpdateStatus(ctx, defaultOpenBankingConnectionAttempt.ID, models.OpenBankingConnectionAttemptStatusPending, nil))

		actual, err := store.OpenBankingConnectionAttemptsGet(ctx, defaultOpenBankingConnectionAttempt.ID)
		require.NoError(t, err)
		require.Equal(t, models.OpenBankingConnectionAttemptStatusPending, actual.Status)
		require.Nil(t, actual.Error)
	})
}

func TestOpenBankingConnectionAttemptsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnectionAttempt(t, ctx, store, defaultOpenBankingConnectionAttempt)
	createOpenBankingConnectionAttempt(t, ctx, store, defaultOpenBankingConnectionAttempt2)

	t.Run("get attempt with all fields", func(t *testing.T) {
		actual, err := store.OpenBankingConnectionAttemptsGet(ctx, defaultOpenBankingConnectionAttempt.ID)
		require.NoError(t, err)
		compareOpenBankingConnectionAttempts(t, defaultOpenBankingConnectionAttempt, *actual)
	})

	t.Run("get attempt with minimal fields", func(t *testing.T) {
		actual, err := store.OpenBankingConnectionAttemptsGet(ctx, defaultOpenBankingConnectionAttempt2.ID)
		require.NoError(t, err)
		compareOpenBankingConnectionAttempts(t, defaultOpenBankingConnectionAttempt2, *actual)
	})

	t.Run("get non-existent attempt", func(t *testing.T) {
		actual, err := store.OpenBankingConnectionAttemptsGet(ctx, uuid.New())
		require.Error(t, err)
		require.Nil(t, actual)
	})
}

func TestOpenBankingConnectionAttemptsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnectionAttempt(t, ctx, store, defaultOpenBankingConnectionAttempt)
	createOpenBankingConnectionAttempt(t, ctx, store, defaultOpenBankingConnectionAttempt2)

	t.Run("list attempts by id", func(t *testing.T) {
		q := NewListPSUOpenBankingConnectionAttemptsQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUOpenBankingConnectionAttemptsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", defaultOpenBankingConnectionAttempt.ID.String())),
		)

		cursor, err := store.OpenBankingConnectionAttemptsList(ctx, defaultOpenBankingConnectionAttempt.PsuID, defaultOpenBankingConnectionAttempt.ConnectorID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		compareOpenBankingConnectionAttempts(t, defaultOpenBankingConnectionAttempt, cursor.Data[0])
	})

	t.Run("list attempts by status", func(t *testing.T) {
		q := NewListPSUOpenBankingConnectionAttemptsQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUOpenBankingConnectionAttemptsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("status", string(models.OpenBankingConnectionAttemptStatusCompleted))),
		)

		cursor, err := store.OpenBankingConnectionAttemptsList(ctx, defaultOpenBankingConnectionAttempt2.PsuID, defaultOpenBankingConnectionAttempt2.ConnectorID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		compareOpenBankingConnectionAttempts(t, defaultOpenBankingConnectionAttempt2, cursor.Data[0])
	})

	t.Run("wrong query operator", func(t *testing.T) {
		q := NewListPSUOpenBankingConnectionAttemptsQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUOpenBankingConnectionAttemptsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("id", "test")),
		)

		cursor, err := store.OpenBankingConnectionAttemptsList(ctx, defaultOpenBankingConnectionAttempt.PsuID, defaultOpenBankingConnectionAttempt.ConnectorID, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("unknown query key", func(t *testing.T) {
		q := NewListPSUOpenBankingConnectionAttemptsQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUOpenBankingConnectionAttemptsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("unknown", "test")),
		)

		cursor, err := store.OpenBankingConnectionAttemptsList(ctx, defaultOpenBankingConnectionAttempt.PsuID, defaultOpenBankingConnectionAttempt.ConnectorID, q)
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
		bankBridge := models.OpenBankingProviderPSU{
			ConnectorID: defaultPSUBankBridge.ConnectorID,
			AccessToken: &models.Token{
				Token:     "access_token_changed",
				ExpiresAt: now.Add(30 * time.Minute).UTC().Time,
			},
			Metadata: map[string]string{
				"changed": "changed",
			},
		}

		require.NoError(t, store.OpenBankingProviderPSUUpsert(ctx, defaultPSU2.ID, bankBridge))

		actual, err := store.OpenBankingProviderPSUGet(ctx, defaultPSU2.ID, defaultPSUBankBridge.ConnectorID)
		require.NoError(t, err)
		// Should update the bank bridge
		require.Equal(t, bankBridge.AccessToken.Token, actual.AccessToken.Token)
		require.Equal(t, bankBridge.Metadata, actual.Metadata)
	})
}

func TestOpenBankingProviderPSUGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridge(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridge)
	createOpenBankingConnection(t, ctx, store, defaultPSU2.ID, bankBridgeConn)

	t.Run("get bank bridge with connections", func(t *testing.T) {
		actual, err := store.OpenBankingProviderPSUGet(ctx, defaultPSU2.ID, defaultPSUBankBridge.ConnectorID)
		require.NoError(t, err)
		compareOpenBankingProviderPSU(t, defaultPSUBankBridge, *actual)
	})

	t.Run("get non-existent bank bridge", func(t *testing.T) {
		actual, err := store.OpenBankingProviderPSUGet(ctx, uuid.New(), defaultPSUBankBridge.ConnectorID)
		require.Error(t, err)
		require.Nil(t, actual)
	})
}

func TestOpenBankingProviderPSUDelete(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridge(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridge)

	t.Run("delete existing bank bridge", func(t *testing.T) {
		require.NoError(t, store.OpenBankingProviderPSUDelete(ctx, defaultPSU2.ID, defaultPSUBankBridge.ConnectorID))

		actual, err := store.OpenBankingProviderPSUGet(ctx, defaultPSU2.ID, defaultPSUBankBridge.ConnectorID)
		require.Error(t, err)
		require.Nil(t, actual)
	})

	t.Run("delete non-existent bank bridge", func(t *testing.T) {
		require.NoError(t, store.OpenBankingProviderPSUDelete(ctx, uuid.New(), defaultPSUBankBridge.ConnectorID))
	})
}

func TestOpenBankingProviderPSUList(t *testing.T) {
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
		q := NewListOpenBankingProviderPSUQuery(
			bunpaginate.NewPaginatedQueryOptions(OpenBankingProviderPSUQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", defaultPSUBankBridge.ConnectorID.String())),
		)

		cursor, err := store.OpenBankingProviderPSUList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotNil(t, cursor.Data[0].AccessToken)
	})

	t.Run("list bank bridges by psu_id", func(t *testing.T) {
		q := NewListOpenBankingProviderPSUQuery(
			bunpaginate.NewPaginatedQueryOptions(OpenBankingProviderPSUQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("psu_id", defaultPSU2.ID.String())),
		)

		cursor, err := store.OpenBankingProviderPSUList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
	})

	t.Run("list bank bridges by metadata", func(t *testing.T) {
		q := NewListOpenBankingProviderPSUQuery(
			bunpaginate.NewPaginatedQueryOptions(OpenBankingProviderPSUQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "bar")),
		)

		cursor, err := store.OpenBankingProviderPSUList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
	})
}

func TestPSUOpenBankingConnectionsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultOpenBankingConnection)

	t.Run("upsert with same connection", func(t *testing.T) {
		connection := models.OpenBankingConnection{
			ConnectorID:   defaultOpenBankingConnection.ConnectorID,
			ConnectionID:  defaultOpenBankingConnection.ConnectionID,
			CreatedAt:     now.Add(-35 * time.Minute).UTC().Time,
			DataUpdatedAt: now.Add(-8 * time.Minute).UTC().Time,
			UpdatedAt:     now.Add(-8 * time.Minute).UTC().Time,
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

		require.NoError(t, store.PSUOpenBankingConnectionsUpsert(ctx, defaultPSU2.ID, connection))

		actual, err := store.PSUOpenBankingConnectionsGet(ctx, defaultPSU2.ID, connection.ConnectorID, connection.ConnectionID)
		require.NoError(t, err)
		// Should update the connection
		require.Equal(t, connection.Status, actual.Status)
		require.Equal(t, connection.UpdatedAt, actual.UpdatedAt)
		require.Equal(t, connection.AccessToken.Token, actual.AccessToken.Token)
		require.Equal(t, connection.Error, actual.Error)
		require.Equal(t, connection.Metadata, actual.Metadata)
	})
}

func TestOpenBankingConnectionsUpdateLastDataUpdate(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultOpenBankingConnection)

	t.Run("update last data update", func(t *testing.T) {
		newUpdatedAt := now.Add(-5 * time.Minute).UTC().Time
		require.NoError(t, store.PSUOpenBankingConnectionsUpdateLastDataUpdate(ctx, defaultPSU2.ID, defaultOpenBankingConnection.ConnectorID, defaultOpenBankingConnection.ConnectionID, newUpdatedAt))

		actual, err := store.PSUOpenBankingConnectionsGet(ctx, defaultPSU2.ID, defaultOpenBankingConnection.ConnectorID, defaultOpenBankingConnection.ConnectionID)
		require.NoError(t, err)
		require.Equal(t, newUpdatedAt, actual.DataUpdatedAt)
	})
}

func TestPSUOpenBankingConnectionsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultOpenBankingConnection)
	createOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultOpenBankingConnection2)

	t.Run("get connection with all fields", func(t *testing.T) {
		actual, err := store.PSUOpenBankingConnectionsGet(ctx, defaultPSU2.ID, defaultOpenBankingConnection.ConnectorID, defaultOpenBankingConnection.ConnectionID)
		require.NoError(t, err)
		compareOpenBankingConnections(t, defaultOpenBankingConnection, *actual)
	})

	t.Run("get connection with error", func(t *testing.T) {
		actual, err := store.PSUOpenBankingConnectionsGet(ctx, defaultPSU2.ID, defaultOpenBankingConnection2.ConnectorID, defaultOpenBankingConnection2.ConnectionID)
		require.NoError(t, err)
		compareOpenBankingConnections(t, defaultOpenBankingConnection2, *actual)
	})

	t.Run("get non-existent connection", func(t *testing.T) {
		actual, err := store.PSUOpenBankingConnectionsGet(ctx, defaultPSU2.ID, defaultOpenBankingConnection.ConnectorID, "non_existent")
		require.Error(t, err)
		require.Nil(t, actual)
	})
}

func TestOpenBankingConnectionsDelete(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultOpenBankingConnection)

	t.Run("delete existing connection", func(t *testing.T) {
		require.NoError(t, store.PSUOpenBankingConnectionsDelete(ctx, defaultPSU2.ID, defaultOpenBankingConnection.ConnectorID, defaultOpenBankingConnection.ConnectionID))

		actual, err := store.PSUOpenBankingConnectionsGet(ctx, defaultPSU2.ID, defaultOpenBankingConnection.ConnectorID, defaultOpenBankingConnection.ConnectionID)
		require.Error(t, err)
		require.Nil(t, actual)
	})

	t.Run("delete non-existent connection", func(t *testing.T) {
		require.NoError(t, store.PSUOpenBankingConnectionsDelete(ctx, defaultPSU2.ID, defaultOpenBankingConnection.ConnectorID, "non_existent"))
	})
}

func TestPSUOpenBankingConnectionsGetFromConnectionID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultOpenBankingConnection)

	t.Run("get connection by connection ID", func(t *testing.T) {
		actual, actualPsuID, err := store.PSUOpenBankingConnectionsGetFromConnectionID(ctx, defaultOpenBankingConnection.ConnectorID, defaultOpenBankingConnection.ConnectionID)
		require.NoError(t, err)
		require.Equal(t, defaultPSU2.ID, actualPsuID)
		compareOpenBankingConnections(t, defaultOpenBankingConnection, *actual)
	})

	t.Run("get non-existent connection by connection ID", func(t *testing.T) {
		actual, actualPsuID, err := store.PSUOpenBankingConnectionsGetFromConnectionID(ctx, defaultOpenBankingConnection.ConnectorID, "non_existent")
		require.Error(t, err)
		require.Equal(t, uuid.Nil, actualPsuID)
		require.Nil(t, actual)
	})
}

func TestOpenBankingConnectionsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultOpenBankingConnection)
	createOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultOpenBankingConnection2)

	t.Run("list connections by connection_id", func(t *testing.T) {
		q := NewListPsuOpenBankingConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuOpenBankingConnectionsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connection_id", defaultOpenBankingConnection.ConnectionID)),
		)

		cursor, err := store.PSUOpenBankingConnectionsList(ctx, defaultPSU2.ID, nil, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		compareOpenBankingConnections(t, defaultOpenBankingConnection, cursor.Data[0])
	})

	t.Run("list connections by status", func(t *testing.T) {
		q := NewListPsuOpenBankingConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuOpenBankingConnectionsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("status", string(models.ConnectionStatusError))),
		)

		cursor, err := store.PSUOpenBankingConnectionsList(ctx, defaultPSU2.ID, nil, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		compareOpenBankingConnections(t, defaultOpenBankingConnection2, cursor.Data[0])
	})

	t.Run("list connections by metadata", func(t *testing.T) {
		q := NewListPsuOpenBankingConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuOpenBankingConnectionsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[conn_foo2]", "conn_bar2")),
		)

		cursor, err := store.PSUOpenBankingConnectionsList(ctx, defaultPSU2.ID, nil, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		compareOpenBankingConnections(t, defaultOpenBankingConnection, cursor.Data[0])
	})

	t.Run("list connections with connector filter", func(t *testing.T) {
		q := NewListPsuOpenBankingConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuOpenBankingConnectionsQuery{}).
				WithPageSize(15),
		)

		cursor, err := store.PSUOpenBankingConnectionsList(ctx, defaultPSU2.ID, &defaultOpenBankingConnection.ConnectorID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
	})

	t.Run("wrong query operator", func(t *testing.T) {
		q := NewListPsuOpenBankingConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuOpenBankingConnectionsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("connection_id", "test")),
		)

		cursor, err := store.PSUOpenBankingConnectionsList(ctx, defaultPSU2.ID, nil, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("unknown query key", func(t *testing.T) {
		q := NewListPsuOpenBankingConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuOpenBankingConnectionsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("unknown", "test")),
		)

		cursor, err := store.PSUOpenBankingConnectionsList(ctx, defaultPSU2.ID, nil, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})
}

func TestOpenBankingProviderPSUGetByPSPUserID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUBankBridge(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridgeWithPSPUserID)
	createPSUBankBridge(t, ctx, store, defaultPSU2.ID, defaultPSUBankBridgeWithPSPUserID2)

	t.Run("get bank bridge by PSPUserID with first connector", func(t *testing.T) {
		actual, err := store.OpenBankingProviderPSUGetByPSPUserID(ctx, *defaultPSUBankBridgeWithPSPUserID.PSPUserID, defaultPSUBankBridgeWithPSPUserID.ConnectorID)
		require.NoError(t, err)
		compareOpenBankingProviderPSU(t, defaultPSUBankBridgeWithPSPUserID, *actual)
	})

	t.Run("get bank bridge by PSPUserID with second connector", func(t *testing.T) {
		actual, err := store.OpenBankingProviderPSUGetByPSPUserID(ctx, *defaultPSUBankBridgeWithPSPUserID2.PSPUserID, defaultPSUBankBridgeWithPSPUserID2.ConnectorID)
		require.NoError(t, err)
		compareOpenBankingProviderPSU(t, defaultPSUBankBridgeWithPSPUserID2, *actual)
	})

	t.Run("get non-existent bank bridge by PSPUserID", func(t *testing.T) {
		actual, err := store.OpenBankingProviderPSUGetByPSPUserID(ctx, "non_existent", defaultPSUBankBridgeWithPSPUserID.ConnectorID)
		require.Error(t, err)
		require.Nil(t, actual)
	})
}

func compareOpenBankingConnectionAttempts(t *testing.T, expected, actual models.OpenBankingConnectionAttempt) {
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

func compareOpenBankingProviderPSU(t *testing.T, expected, actual models.OpenBankingProviderPSU) {
	require.Equal(t, expected.ConnectorID, actual.ConnectorID)
	require.Equal(t, expected.Metadata, actual.Metadata)

	if expected.AccessToken != nil {
		require.NotNil(t, actual.AccessToken)
		require.Equal(t, expected.AccessToken.Token, actual.AccessToken.Token)
		require.Equal(t, expected.AccessToken.ExpiresAt, actual.AccessToken.ExpiresAt)
	} else {
		require.Nil(t, actual.AccessToken)
	}
}

func compareOpenBankingConnections(t *testing.T, expected, actual models.OpenBankingConnection) {
	require.Equal(t, expected.ConnectorID, actual.ConnectorID)
	require.Equal(t, expected.ConnectionID, actual.ConnectionID)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.DataUpdatedAt, actual.DataUpdatedAt)
	require.Equal(t, expected.UpdatedAt, actual.UpdatedAt)
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.Error, actual.Error)
	require.Equal(t, expected.Metadata, actual.Metadata)

	if expected.AccessToken != nil && actual.AccessToken != nil {
		require.NotNil(t, actual.AccessToken)
		require.Equal(t, expected.AccessToken.Token, actual.AccessToken.Token)
		require.Equal(t, expected.AccessToken.ExpiresAt, actual.AccessToken.ExpiresAt)
	} else {
		require.Nil(t, actual.AccessToken)
	}
}
