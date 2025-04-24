package storage

import (
	"context"
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStorageErrorHandling(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	ctx := context.Background()

	t.Run("AccountsGet with non-existent ID", func(t *testing.T) {
		nonExistentID := models.AccountID(uuid.New())
		account, err := store.AccountsGet(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, account)
	})

	t.Run("PaymentsGet with non-existent ID", func(t *testing.T) {
		nonExistentID := models.PaymentID(uuid.New())
		payment, err := store.PaymentsGet(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, payment)
	})

	t.Run("BankAccountsGet with non-existent ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		bankAccount, err := store.BankAccountsGet(ctx, nonExistentID, false)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, bankAccount)
	})

	t.Run("PoolsGet with non-existent ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		pool, err := store.PoolsGet(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, pool)
	})

	t.Run("ConnectorsGet with non-existent ID", func(t *testing.T) {
		nonExistentID := models.ConnectorID{
			Reference: uuid.New().String(),
			Provider:  "test",
		}
		connector, err := store.ConnectorsGet(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, connector)
	})
}
