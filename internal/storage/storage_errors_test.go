package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStorageErrorHandling(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	ctx := context.Background()

	t.Run("AccountsGet with non-existent ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		account, err := store.AccountsGet(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, account)
	})

	t.Run("PaymentsGet with non-existent ID", func(t *testing.T) {
		nonExistentID := uuid.New()
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

	t.Run("BalancesGetAt with non-existent ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		balances, err := store.BalancesGetAt(ctx, nonExistentID, time.Now())
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, balances)
	})

	t.Run("ConnectorsGet with non-existent ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		connector, err := store.ConnectorsGet(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, connector)
	})
}
