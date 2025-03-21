package storage

import (
	"context"
	"testing"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var (
	defaultTasksTree = models.ConnectorTasksTree{
		{
			TaskType:     models.TASK_FETCH_ACCOUNTS,
			Name:         "fetch_accounts",
			Periodically: true,
			NextTasks: []models.ConnectorTaskTree{
				{
					TaskType:     models.TASK_FETCH_PAYMENTS,
					Name:         "fetch_payments",
					Periodically: true,
					NextTasks:    []models.ConnectorTaskTree{},
				},
				{
					TaskType:     models.TASK_FETCH_BALANCES,
					Name:         "fetch_balances",
					Periodically: true,
					NextTasks:    []models.ConnectorTaskTree{},
				},
			},
		},
		{
			TaskType:     models.TASK_FETCH_EXTERNAL_ACCOUNTS,
			Name:         "fetch_beneficiaries",
			Periodically: true,
			NextTasks:    []models.ConnectorTaskTree{},
		},
	}

	defaultTasksTree2 = models.ConnectorTasksTree{
		{
			TaskType:     models.TASK_FETCH_ACCOUNTS,
			Name:         "fetch_accounts",
			Periodically: true,
			NextTasks: []models.ConnectorTaskTree{
				{
					TaskType:     models.TASK_FETCH_BALANCES,
					Name:         "fetch_balances",
					Periodically: true,
					NextTasks:    []models.ConnectorTaskTree{},
				},
				{
					TaskType:     models.TASK_FETCH_PAYMENTS,
					Name:         "fetch_payments",
					Periodically: true,
					NextTasks:    []models.ConnectorTaskTree{},
				},
				{
					TaskType:     models.TASK_FETCH_EXTERNAL_ACCOUNTS,
					Name:         "fetch_recipients",
					Periodically: true,
					NextTasks:    []models.ConnectorTaskTree{},
				},
			},
		},
	}
)

func upsertTasksTree(t *testing.T, ctx context.Context, storage Storage, connectorID models.ConnectorID, tasksTree []models.ConnectorTaskTree) {
	require.NoError(t, storage.ConnectorTasksTreeUpsert(ctx, connectorID, tasksTree))
}

func TestConnectorTasksTreeUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertTasksTree(t, ctx, store, defaultConnector.ID, defaultTasksTree)

	t.Run("upsert with unknown connector id", func(t *testing.T) {
		require.Error(t, store.ConnectorTasksTreeUpsert(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}, defaultTasksTree2))
	})

	t.Run("upsert with same connector id", func(t *testing.T) {
		upsertTasksTree(t, ctx, store, defaultConnector.ID, defaultTasksTree2)

		tasks, err := store.ConnectorTasksTreeGet(ctx, defaultConnector.ID)
		require.NoError(t, err)
		require.Equal(t, defaultTasksTree2, *tasks)
	})
}

func TestConnectorTasksTreeGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertTasksTree(t, ctx, store, defaultConnector.ID, defaultTasksTree)

	t.Run("get tasks", func(t *testing.T) {
		tasks, err := store.ConnectorTasksTreeGet(ctx, defaultConnector.ID)
		require.NoError(t, err)
		require.Equal(t, defaultTasksTree, *tasks)
	})

	t.Run("get tasks with unknown connector id", func(t *testing.T) {
		_, err := store.ConnectorTasksTreeGet(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		})
		require.Error(t, err)
	})
}

func TestConnectorTasksTreeDeleteFromConnectorID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	defer store.Close()

	upsertConnector(t, ctx, store, defaultConnector)
	upsertConnector(t, ctx, store, defaultConnector2)
	upsertTasksTree(t, ctx, store, defaultConnector.ID, defaultTasksTree)
	upsertTasksTree(t, ctx, store, defaultConnector2.ID, defaultTasksTree2)

	t.Run("delete tasks with unknown connector id", func(t *testing.T) {
		require.NoError(t, store.ConnectorTasksTreeDeleteFromConnectorID(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}))

		tasks, err := store.ConnectorTasksTreeGet(ctx, defaultConnector.ID)
		require.NoError(t, err)
		require.Equal(t, defaultTasksTree, *tasks)

		tasks, err = store.ConnectorTasksTreeGet(ctx, defaultConnector2.ID)
		require.NoError(t, err)
		require.Equal(t, defaultTasksTree2, *tasks)
	})

	t.Run("delete tasks", func(t *testing.T) {
		require.NoError(t, store.ConnectorTasksTreeDeleteFromConnectorID(ctx, defaultConnector.ID))

		_, err := store.ConnectorTasksTreeGet(ctx, defaultConnector.ID)
		require.Error(t, err)

		tasks, err := store.ConnectorTasksTreeGet(ctx, defaultConnector2.ID)
		require.NoError(t, err)
		require.Equal(t, defaultTasksTree2, *tasks)
	})
}
