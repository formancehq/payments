package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

var (
	ErrValidation        = errors.New("validation error")
	ErrNotFound          = errors.New("not found")
	ErrDuplicateKeyValue = errors.New("object already exists")
	// Don't want to expose the internal that is a foreign key violation to the
	// client through the API
	ErrForeignKeyViolation = errors.New("value not found")
)

var FKViolationColumn = []string{
	"connector_id",
	"bank_account_id",
	"payment_id",
	"pool_id",
	"schedule_id",
	"payment_initiation_id",
	"payment_initiation_reversal_id",
}

func e(msg string, err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicateKeyValue
	}

	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		for _, column := range FKViolationColumn {
			if strings.Contains(pgErr.ConstraintName, column) {
				return fmt.Errorf("%s: %w", column, ErrForeignKeyViolation)
			}
		}

		return ErrForeignKeyViolation
	}

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	return fmt.Errorf("%s: %w", msg, err)
}

// meant to be called in defer block
func rollbackOnTxError(ctx context.Context, tx *bun.Tx, err error) {
	if err == nil {
		return
	}

	if rollbackErr := tx.Rollback(); rollbackErr != nil {
		logging.FromContext(ctx).WithField("original_error", err.Error()).Errorf("failed to rollback transaction: %w", rollbackErr)
	}
}
