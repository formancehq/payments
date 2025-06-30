package services

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/pkg/errors"
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("not found")
)

type storageError struct {
	err error
	msg string
}

func (e *storageError) Error() string {
	return fmt.Sprintf("%s: %s", e.msg, e.err)
}

func (e *storageError) Is(err error) bool {
	_, ok := err.(*storageError)
	return ok
}

func (e *storageError) Unwrap() error {
	return e.err
}

func newStorageError(err error, msg string) error {
	if err == nil {
		return nil
	}
	return &storageError{
		err: err,
		msg: msg,
	}
}

func handleEngineErrors(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, engine.ErrValidation), errors.Is(err, models.ErrValidation):
		return errorsutils.NewWrappedError(
			err,
			ErrValidation,
		)
	case errors.Is(err, engine.ErrNotFound):
		return fmt.Errorf("%w: %w", err, ErrNotFound)
	default:
		return err
	}
}
