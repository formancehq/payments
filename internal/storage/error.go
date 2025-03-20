package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

var (
	ErrValidation          = errors.New("validation error")
	ErrNotFound            = errors.New("not found")
	ErrDuplicateKeyValue   = errors.New("duplicate key value")
	ErrForeignKeyViolation = errors.New("foreign key constraint violation: referenced row missing")
)

func e(msg string, err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicateKeyValue
	}

	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrForeignKeyViolation
	}

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	return fmt.Errorf("%s: %w", msg, err)
}

// meant to be called in defer block
func rollbackOnTxError(ctx context.Context, tx bun.Tx, err error) {
	if err == nil {
		return
	}

	if rollbackErr := tx.Rollback(); rollbackErr != nil {
		logging.FromContext(ctx).WithField("original_error", err.Error()).Errorf("failed to rollback transaction: %w", rollbackErr)
	}
}
