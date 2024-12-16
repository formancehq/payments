package activities

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/temporal"
)

const (
	ErrTypeStorage         = "STORAGE"
	ErrTypeDefault         = "DEFAULT"
	ErrTypeInvalidArgument = "INVALID_ARGUMENT"
	ErrTypeUnimplemented   = "UNIMPLEMENTED"
)

func temporalPluginError(err error) error {
	switch {
	// Do not retry the following errors
	case errors.Is(err, plugins.ErrNotImplemented):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeUnimplemented, err)
	case errors.Is(err, plugins.ErrInvalidClientRequest):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeInvalidArgument, err)
	case errors.Is(err, plugins.ErrCurrencyNotSupported):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeInvalidArgument, err)

	// Retry the following errors
	case errors.Is(err, plugins.ErrNotYetInstalled):
		// We want to retry in case of not installed
		return temporal.NewApplicationErrorWithCause(err.Error(), ErrTypeDefault, err)
	default:
		return temporal.NewApplicationErrorWithCause(err.Error(), ErrTypeDefault, err)
	}
}

func temporalStorageError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, storage.ErrNotFound),
		errors.Is(err, storage.ErrDuplicateKeyValue),
		errors.Is(err, storage.ErrValidation),
		errors.Is(err, storage.ErrForeignKeyViolation):
		// Do not retry these errors
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeStorage, err)
	default:
		return temporal.NewApplicationErrorWithCause(err.Error(), ErrTypeStorage, err)
	}
}
