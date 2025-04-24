package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("nil error", func(t *testing.T) {
		t.Parallel()
		err := e("test message", nil)
		require.Nil(t, err)
	})

	t.Run("duplicate key error", func(t *testing.T) {
		t.Parallel()
		pgErr := &pgconn.PgError{
			Code: "23505", // Unique violation code
		}
		err := e("test message", pgErr)
		require.ErrorIs(t, err, ErrDuplicateKeyValue)
	})

	t.Run("foreign key violation", func(t *testing.T) {
		t.Parallel()
		for _, column := range FKViolationColumn {
			pgErr := &pgconn.PgError{
				Code:           "23503", // Foreign key violation code
				ConstraintName: fmt.Sprintf("fk_%s_constraint", column),
			}
			err := e("test message", pgErr)
			require.ErrorIs(t, err, ErrForeignKeyViolation)
			require.True(t, strings.Contains(err.Error(), column))
		}

		pgErr := &pgconn.PgError{
			Code:           "23503",
			ConstraintName: "unknown_constraint",
		}
		err := e("test message", pgErr)
		require.ErrorIs(t, err, ErrForeignKeyViolation)
	})

	t.Run("not found error", func(t *testing.T) {
		t.Parallel()
		err := e("test message", sql.ErrNoRows)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("generic error", func(t *testing.T) {
		t.Parallel()
		origErr := errors.New("original error")
		err := e("test message", origErr)
		require.ErrorIs(t, err, origErr)
		require.Contains(t, err.Error(), "test message")
	})
}

func TestRollbackOnTxError(t *testing.T) {
	t.Parallel()
	
	s := newStore(t)
	ctx := context.Background()

	t.Run("no error", func(t *testing.T) {
		t.Parallel()
		
		db := s.(interface{ GetDB() *bun.DB }).GetDB()
		tx, err := db.Begin()
		require.NoError(t, err)
		
		rollbackOnTxError(ctx, &tx, nil)
		
		err = tx.Commit()
		require.NoError(t, err)
	})
}
