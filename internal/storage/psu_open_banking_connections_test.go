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
	defaultPSUOpenBankingConnectionAttempt = models.PSUOpenBankingConnectionAttempt{
		ID:          uuid.New(),
		PsuID:       defaultPSU2.ID,
		ConnectorID: defaultConnector.ID,
		CreatedAt:   now.Add(-60 * time.Minute).UTC().Time,
		Status:      models.PSUOpenBankingConnectionAttemptStatusPending,
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

	defaultPSUOpenBankingConnectionAttempt2 = models.PSUOpenBankingConnectionAttempt{
		ID:          uuid.New(),
		PsuID:       defaultPSU2.ID,
		ConnectorID: defaultConnector.ID,
		CreatedAt:   now.Add(-30 * time.Minute).UTC().Time,
		Status:      models.PSUOpenBankingConnectionAttemptStatusCompleted,
		State: models.CallbackState{
			Randomized: "random456",
			AttemptID:  uuid.New(),
		},
	}

	defaultObProviderPSU = models.OpenBankingProviderPSU{
		ConnectorID: defaultConnector.ID,
		AccessToken: &models.Token{
			Token:     "access_token_123",
			ExpiresAt: now.Add(60 * time.Minute).UTC().Time,
		},
		Metadata: map[string]string{
			"foo": "bar",
		},
	}

	openBankingConn = models.PSUOpenBankingConnection{
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
	}

	defaultPSUOpenBanking2 = models.OpenBankingProviderPSU{
		ConnectorID: defaultConnector2.ID,
		AccessToken: &models.Token{
			Token:     "access_token_123",
			ExpiresAt: now.Add(60 * time.Minute).UTC().Time,
		},
	}

	defaultPSUOpenBankingWithPSPUserID = models.OpenBankingProviderPSU{
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

	defaultPSUOpenBankingWithPSPUserID2 = models.OpenBankingProviderPSU{
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

	defaultPSUOpenBankingConnection = models.PSUOpenBankingConnection{
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

	defaultPSUOpenBankingConnection2 = models.PSUOpenBankingConnection{
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

func createPSUOpenBankingConnectionAttempt(t *testing.T, ctx context.Context, storage Storage, attempt models.PSUOpenBankingConnectionAttempt) {
	require.NoError(t, storage.PSUOpenBankingConnectionAttemptsUpsert(ctx, attempt))
}

func createOpenBankingProviderPSU(t *testing.T, ctx context.Context, storage Storage, psuID uuid.UUID, bankBridge models.OpenBankingProviderPSU) {
	require.NoError(t, storage.OpenBankingProviderPSUUpsert(ctx, psuID, bankBridge))
}

func createPSUOpenBankingConnection(t *testing.T, ctx context.Context, storage Storage, psuID uuid.UUID, connection models.PSUOpenBankingConnection) {
	require.NoError(t, storage.PSUOpenBankingConnectionsUpsert(ctx, psuID, connection))
}

func TestPSUOpenBankingConnectionAttemptsUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUOpenBankingConnectionAttempt(t, ctx, store, defaultPSUOpenBankingConnectionAttempt)

	t.Run("upsert with same id", func(t *testing.T) {
		attempt := models.PSUOpenBankingConnectionAttempt{
			ID:          defaultPSUOpenBankingConnectionAttempt.ID,
			PsuID:       defaultPSUOpenBankingConnectionAttempt.PsuID,
			ConnectorID: defaultPSUOpenBankingConnectionAttempt.ConnectorID,
			CreatedAt:   now.Add(-50 * time.Minute).UTC().Time,
			Status:      models.PSUOpenBankingConnectionAttemptStatusExited,
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

		require.NoError(t, store.PSUOpenBankingConnectionAttemptsUpsert(ctx, attempt))

		actual, err := store.PSUOpenBankingConnectionAttemptsGet(ctx, defaultPSUOpenBankingConnectionAttempt.ID)
		require.NoError(t, err)
		// Should update the attempt with new values
		require.Equal(t, attempt.Status, actual.Status)
		require.Equal(t, attempt.State, actual.State)
		require.NotNil(t, actual.ClientRedirectURL)
		require.Equal(t, *defaultPSUOpenBankingConnectionAttempt.ClientRedirectURL, *actual.ClientRedirectURL)
		require.Equal(t, attempt.TemporaryToken.Token, actual.TemporaryToken.Token)
		require.Equal(t, attempt.Error, actual.Error)
	})
}

func TestPSUOpenBankingConnectionAttemptsUpdateStatus(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUOpenBankingConnectionAttempt(t, ctx, store, defaultPSUOpenBankingConnectionAttempt)

	t.Run("update status to completed", func(t *testing.T) {
		errMsg := pointer.For("Successfully completed")
		require.NoError(t, store.PSUOpenBankingConnectionAttemptsUpdateStatus(ctx, defaultPSUOpenBankingConnectionAttempt.ID, models.PSUOpenBankingConnectionAttemptStatusCompleted, errMsg))

		actual, err := store.PSUOpenBankingConnectionAttemptsGet(ctx, defaultPSUOpenBankingConnectionAttempt.ID)
		require.NoError(t, err)
		require.Equal(t, models.PSUOpenBankingConnectionAttemptStatusCompleted, actual.Status)
		require.Equal(t, errMsg, actual.Error)
	})

	t.Run("update status to failed", func(t *testing.T) {
		errMsg := pointer.For("Connection failed")
		require.NoError(t, store.PSUOpenBankingConnectionAttemptsUpdateStatus(ctx, defaultPSUOpenBankingConnectionAttempt.ID, models.PSUOpenBankingConnectionAttemptStatusExited, errMsg))

		actual, err := store.PSUOpenBankingConnectionAttemptsGet(ctx, defaultPSUOpenBankingConnectionAttempt.ID)
		require.NoError(t, err)
		require.Equal(t, models.PSUOpenBankingConnectionAttemptStatusExited, actual.Status)
		require.Equal(t, errMsg, actual.Error)
	})

	t.Run("update status with nil error", func(t *testing.T) {
		require.NoError(t, store.PSUOpenBankingConnectionAttemptsUpdateStatus(ctx, defaultPSUOpenBankingConnectionAttempt.ID, models.PSUOpenBankingConnectionAttemptStatusPending, nil))

		actual, err := store.PSUOpenBankingConnectionAttemptsGet(ctx, defaultPSUOpenBankingConnectionAttempt.ID)
		require.NoError(t, err)
		require.Equal(t, models.PSUOpenBankingConnectionAttemptStatusPending, actual.Status)
		require.Nil(t, actual.Error)
	})
}

func TestPSUOpenBankingConnectionAttemptsGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUOpenBankingConnectionAttempt(t, ctx, store, defaultPSUOpenBankingConnectionAttempt)
	createPSUOpenBankingConnectionAttempt(t, ctx, store, defaultPSUOpenBankingConnectionAttempt2)

	t.Run("get attempt with all fields", func(t *testing.T) {
		actual, err := store.PSUOpenBankingConnectionAttemptsGet(ctx, defaultPSUOpenBankingConnectionAttempt.ID)
		require.NoError(t, err)
		comparePSUOpenBankingConnectionAttempts(t, defaultPSUOpenBankingConnectionAttempt, *actual)
	})

	t.Run("get attempt with minimal fields", func(t *testing.T) {
		actual, err := store.PSUOpenBankingConnectionAttemptsGet(ctx, defaultPSUOpenBankingConnectionAttempt2.ID)
		require.NoError(t, err)
		comparePSUOpenBankingConnectionAttempts(t, defaultPSUOpenBankingConnectionAttempt2, *actual)
	})

	t.Run("get non-existent attempt", func(t *testing.T) {
		actual, err := store.PSUOpenBankingConnectionAttemptsGet(ctx, uuid.New())
		require.Error(t, err)
		require.Nil(t, actual)
	})
}

func TestPSUOpenBankingConnectionAttemptsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUOpenBankingConnectionAttempt(t, ctx, store, defaultPSUOpenBankingConnectionAttempt)
	createPSUOpenBankingConnectionAttempt(t, ctx, store, defaultPSUOpenBankingConnectionAttempt2)

	t.Run("list attempts by id", func(t *testing.T) {
		q := NewListPSUOpenBankingConnectionAttemptsQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUOpenBankingConnectionAttemptsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", defaultPSUOpenBankingConnectionAttempt.ID.String())),
		)

		cursor, err := store.PSUOpenBankingConnectionAttemptsList(ctx, defaultPSUOpenBankingConnectionAttempt.PsuID, defaultPSUOpenBankingConnectionAttempt.ConnectorID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePSUOpenBankingConnectionAttempts(t, defaultPSUOpenBankingConnectionAttempt, cursor.Data[0])
	})

	t.Run("list attempts by status", func(t *testing.T) {
		q := NewListPSUOpenBankingConnectionAttemptsQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUOpenBankingConnectionAttemptsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("status", string(models.PSUOpenBankingConnectionAttemptStatusCompleted))),
		)

		cursor, err := store.PSUOpenBankingConnectionAttemptsList(ctx, defaultPSUOpenBankingConnectionAttempt2.PsuID, defaultPSUOpenBankingConnectionAttempt2.ConnectorID, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePSUOpenBankingConnectionAttempts(t, defaultPSUOpenBankingConnectionAttempt2, cursor.Data[0])
	})

	t.Run("wrong query operator", func(t *testing.T) {
		q := NewListPSUOpenBankingConnectionAttemptsQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUOpenBankingConnectionAttemptsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("id", "test")),
		)

		cursor, err := store.PSUOpenBankingConnectionAttemptsList(ctx, defaultPSUOpenBankingConnectionAttempt.PsuID, defaultPSUOpenBankingConnectionAttempt.ConnectorID, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("unknown query key", func(t *testing.T) {
		q := NewListPSUOpenBankingConnectionAttemptsQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUOpenBankingConnectionAttemptsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("unknown", "test")),
		)

		cursor, err := store.PSUOpenBankingConnectionAttemptsList(ctx, defaultPSUOpenBankingConnectionAttempt.PsuID, defaultPSUOpenBankingConnectionAttempt.ConnectorID, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})
}

func TestPSUOpenBankingUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingProviderPSU(t, ctx, store, defaultPSU2.ID, defaultObProviderPSU)

	t.Run("upsert with same psu and connector", func(t *testing.T) {
		obProviderPSU := models.OpenBankingProviderPSU{
			ConnectorID: defaultObProviderPSU.ConnectorID,
			AccessToken: &models.Token{
				Token:     "access_token_changed",
				ExpiresAt: now.Add(30 * time.Minute).UTC().Time,
			},
			Metadata: map[string]string{
				"changed": "changed",
			},
		}

		require.NoError(t, store.OpenBankingProviderPSUUpsert(ctx, defaultPSU2.ID, obProviderPSU))

		actual, err := store.OpenBankingProviderPSUGet(ctx, defaultPSU2.ID, defaultObProviderPSU.ConnectorID)
		require.NoError(t, err)
		// Should update the provider psu
		require.Equal(t, obProviderPSU.AccessToken.Token, actual.AccessToken.Token)
		require.Equal(t, obProviderPSU.Metadata, actual.Metadata)
	})
}

func TestPSUOpenBankingGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingProviderPSU(t, ctx, store, defaultPSU2.ID, defaultObProviderPSU)
	createPSUOpenBankingConnection(t, ctx, store, defaultPSU2.ID, openBankingConn)

	t.Run("get ob provider psu with connections", func(t *testing.T) {
		actual, err := store.OpenBankingProviderPSUGet(ctx, defaultPSU2.ID, defaultObProviderPSU.ConnectorID)
		require.NoError(t, err)
		compareOBProviderPSU(t, defaultObProviderPSU, *actual)
	})

	t.Run("get non-existent ob provider psu", func(t *testing.T) {
		actual, err := store.OpenBankingProviderPSUGet(ctx, uuid.New(), defaultObProviderPSU.ConnectorID)
		require.Error(t, err)
		require.Nil(t, actual)
	})
}

func TestPSUOpenBankingDelete(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingProviderPSU(t, ctx, store, defaultPSU2.ID, defaultObProviderPSU)

	t.Run("delete existing open banking provider psu", func(t *testing.T) {
		require.NoError(t, store.OpenBankingProviderPSUDelete(ctx, defaultPSU2.ID, defaultObProviderPSU.ConnectorID))

		actual, err := store.OpenBankingProviderPSUGet(ctx, defaultPSU2.ID, defaultObProviderPSU.ConnectorID)
		require.Error(t, err)
		require.Nil(t, actual)
	})

	t.Run("delete non-existent open banking provider psu", func(t *testing.T) {
		require.NoError(t, store.OpenBankingProviderPSUDelete(ctx, uuid.New(), defaultObProviderPSU.ConnectorID))
	})
}

func TestPSUOpenBankingList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingProviderPSU(t, ctx, store, defaultPSU2.ID, defaultObProviderPSU)
	createOpenBankingProviderPSU(t, ctx, store, defaultPSU2.ID, defaultPSUOpenBanking2)

	t.Run("list bank bridges by connector_id", func(t *testing.T) {
		q := NewListOpenBankingProviderPSUQuery(
			bunpaginate.NewPaginatedQueryOptions(OpenBankingProviderPSUQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connector_id", defaultObProviderPSU.ConnectorID.String())),
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
	createPSUOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultPSUOpenBankingConnection)

	t.Run("upsert with same connection", func(t *testing.T) {
		connection := models.PSUOpenBankingConnection{
			ConnectorID:   defaultPSUOpenBankingConnection.ConnectorID,
			ConnectionID:  defaultPSUOpenBankingConnection.ConnectionID,
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

		require.NoError(t, store.PSUOpenBankingConnectionsUpsert(ctx, defaultPSU2.ID, connection))

		actual, err := store.PSUOpenBankingConnectionsGet(ctx, defaultPSU2.ID, connection.ConnectorID, connection.ConnectionID)
		require.NoError(t, err)
		// Should update the connection
		require.Equal(t, connection.Status, actual.Status)
		require.Equal(t, connection.AccessToken.Token, actual.AccessToken.Token)
		require.Equal(t, connection.Error, actual.Error)
		require.Equal(t, connection.Metadata, actual.Metadata)
	})
}

func TestPSUOpenBankingConnectionsUpdateLastDataUpdate(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultPSUOpenBankingConnection)

	t.Run("update last data update", func(t *testing.T) {
		newUpdatedAt := now.Add(-5 * time.Minute).UTC().Time
		require.NoError(t, store.PSUOpenBankingConnectionsUpdateLastDataUpdate(ctx, defaultPSU2.ID, defaultPSUOpenBankingConnection.ConnectorID, defaultPSUOpenBankingConnection.ConnectionID, newUpdatedAt))

		actual, err := store.PSUOpenBankingConnectionsGet(ctx, defaultPSU2.ID, defaultPSUOpenBankingConnection.ConnectorID, defaultPSUOpenBankingConnection.ConnectionID)
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
	createPSUOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultPSUOpenBankingConnection)
	createPSUOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultPSUOpenBankingConnection2)

	t.Run("get connection with all fields", func(t *testing.T) {
		actual, err := store.PSUOpenBankingConnectionsGet(ctx, defaultPSU2.ID, defaultPSUOpenBankingConnection.ConnectorID, defaultPSUOpenBankingConnection.ConnectionID)
		require.NoError(t, err)
		comparePSUOpenBankingConnections(t, defaultPSUOpenBankingConnection, *actual)
	})

	t.Run("get connection with error", func(t *testing.T) {
		actual, err := store.PSUOpenBankingConnectionsGet(ctx, defaultPSU2.ID, defaultPSUOpenBankingConnection2.ConnectorID, defaultPSUOpenBankingConnection2.ConnectionID)
		require.NoError(t, err)
		comparePSUOpenBankingConnections(t, defaultPSUOpenBankingConnection2, *actual)
	})

	t.Run("get non-existent connection", func(t *testing.T) {
		actual, err := store.PSUOpenBankingConnectionsGet(ctx, defaultPSU2.ID, defaultPSUOpenBankingConnection.ConnectorID, "non_existent")
		require.Error(t, err)
		require.Nil(t, actual)
	})
}

func TestPSUOpenBankingConnectionsDelete(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultPSUOpenBankingConnection)

	t.Run("delete existing connection", func(t *testing.T) {
		require.NoError(t, store.PSUOpenBankingConnectionsDelete(ctx, defaultPSU2.ID, defaultPSUOpenBankingConnection.ConnectorID, defaultPSUOpenBankingConnection.ConnectionID))

		actual, err := store.PSUOpenBankingConnectionsGet(ctx, defaultPSU2.ID, defaultPSUOpenBankingConnection.ConnectorID, defaultPSUOpenBankingConnection.ConnectionID)
		require.Error(t, err)
		require.Nil(t, actual)
	})

	t.Run("delete non-existent connection", func(t *testing.T) {
		require.NoError(t, store.PSUOpenBankingConnectionsDelete(ctx, defaultPSU2.ID, defaultPSUOpenBankingConnection.ConnectorID, "non_existent"))
	})
}

func TestPSUOpenBankingConnectionsGetFromConnectionID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultPSUOpenBankingConnection)

	t.Run("get connection by connection ID", func(t *testing.T) {
		actual, actualPsuID, err := store.PSUOpenBankingConnectionsGetFromConnectionID(ctx, defaultPSUOpenBankingConnection.ConnectorID, defaultPSUOpenBankingConnection.ConnectionID)
		require.NoError(t, err)
		require.Equal(t, defaultPSU2.ID, actualPsuID)
		comparePSUOpenBankingConnections(t, defaultPSUOpenBankingConnection, *actual)
	})

	t.Run("get non-existent connection by connection ID", func(t *testing.T) {
		actual, actualPsuID, err := store.PSUOpenBankingConnectionsGetFromConnectionID(ctx, defaultPSUOpenBankingConnection.ConnectorID, "non_existent")
		require.Error(t, err)
		require.Equal(t, uuid.Nil, actualPsuID)
		require.Nil(t, actual)
	})
}

func TestPSUOpenBankingConnectionsList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	createPSU(t, ctx, store, defaultPSU2)
	createPSUOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultPSUOpenBankingConnection)
	createPSUOpenBankingConnection(t, ctx, store, defaultPSU2.ID, defaultPSUOpenBankingConnection2)

	t.Run("list connections by connection_id", func(t *testing.T) {
		q := NewListPsuOpenBankingConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuOpenBankingConnectionsQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("connection_id", defaultPSUOpenBankingConnection.ConnectionID)),
		)

		cursor, err := store.PSUOpenBankingConnectionsList(ctx, defaultPSU2.ID, nil, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		comparePSUOpenBankingConnections(t, defaultPSUOpenBankingConnection, cursor.Data[0])
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
		comparePSUOpenBankingConnections(t, defaultPSUOpenBankingConnection2, cursor.Data[0])
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
		comparePSUOpenBankingConnections(t, defaultPSUOpenBankingConnection, cursor.Data[0])
	})

	t.Run("list connections with connector filter", func(t *testing.T) {
		q := NewListPsuOpenBankingConnectionsQuery(
			bunpaginate.NewPaginatedQueryOptions(PsuOpenBankingConnectionsQuery{}).
				WithPageSize(15),
		)

		cursor, err := store.PSUOpenBankingConnectionsList(ctx, defaultPSU2.ID, &defaultPSUOpenBankingConnection.ConnectorID, q)
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

func TestPSUOpenBankingGetByPSPUserID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	createPSU(t, ctx, store, defaultPSU2)
	createOpenBankingProviderPSU(t, ctx, store, defaultPSU2.ID, defaultPSUOpenBankingWithPSPUserID)
	createOpenBankingProviderPSU(t, ctx, store, defaultPSU2.ID, defaultPSUOpenBankingWithPSPUserID2)

	t.Run("get ob provider psu by PSPUserID with first connector", func(t *testing.T) {
		actual, err := store.OpenBankingProviderPSUGetByPSPUserID(ctx, *defaultPSUOpenBankingWithPSPUserID.PSPUserID, defaultPSUOpenBankingWithPSPUserID.ConnectorID)
		require.NoError(t, err)
		compareOBProviderPSU(t, defaultPSUOpenBankingWithPSPUserID, *actual)
	})

	t.Run("get ob provider psu by PSPUserID with second connector", func(t *testing.T) {
		actual, err := store.OpenBankingProviderPSUGetByPSPUserID(ctx, *defaultPSUOpenBankingWithPSPUserID2.PSPUserID, defaultPSUOpenBankingWithPSPUserID2.ConnectorID)
		require.NoError(t, err)
		compareOBProviderPSU(t, defaultPSUOpenBankingWithPSPUserID2, *actual)
	})

	t.Run("get non-existent ob provider psu by PSPUserID", func(t *testing.T) {
		actual, err := store.OpenBankingProviderPSUGetByPSPUserID(ctx, "non_existent", defaultPSUOpenBankingWithPSPUserID.ConnectorID)
		require.Error(t, err)
		require.Nil(t, actual)
	})
}

func comparePSUOpenBankingConnectionAttempts(t *testing.T, expected, actual models.PSUOpenBankingConnectionAttempt) {
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

func compareOBProviderPSU(t *testing.T, expected, actual models.OpenBankingProviderPSU) {
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

func comparePSUOpenBankingConnections(t *testing.T, expected, actual models.PSUOpenBankingConnection) {
	require.Equal(t, expected.ConnectorID, actual.ConnectorID)
	require.Equal(t, expected.ConnectionID, actual.ConnectionID)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.DataUpdatedAt, actual.DataUpdatedAt)
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
