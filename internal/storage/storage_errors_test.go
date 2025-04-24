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

	t.Run("GetAccount with non-existent ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		account, err := store.GetAccount(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, account)
	})

	t.Run("GetPayment with non-existent ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		payment, err := store.GetPayment(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, payment)
	})

	t.Run("GetBankAccount with non-existent ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		bankAccount, err := store.GetBankAccount(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, bankAccount)
	})

	t.Run("GetBalance with non-existent ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		balance, err := store.GetBalance(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, balance)
	})

	t.Run("GetConnector with non-existent ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		connector, err := store.GetConnector(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
		require.Nil(t, connector)
	})
}
