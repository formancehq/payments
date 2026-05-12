package activities

import (
	"context"
	"errors"

	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"go.temporal.io/sdk/temporal"
)

const (
	ErrTypeStorage         = "STORAGE"
	ErrTypeDefault         = "DEFAULT"
	ErrTypeInvalidArgument = "INVALID_ARGUMENT"
	ErrTypeRateLimited     = "RATE_LIMITED"
	ErrTypeTimeout         = "TIMEOUT"
	ErrTypeRetryAfter      = "RETRY_AFTER"
	ErrTypeUnimplemented   = "UNIMPLEMENTED"
)

func (a Activities) temporalPluginError(ctx context.Context, err error) error {
	return a.temporalPluginErrorCheck(ctx, err, false)
}

func (a Activities) temporalPluginPollingError(ctx context.Context, err error, periodic bool) error {
	return a.temporalPluginErrorCheck(ctx, err, periodic)
}

func (a Activities) temporalPluginErrorCheck(ctx context.Context, err error, isPeriodic bool) error {
	// Since typed errors are discard with temporal when returning from an activity,
	// we need only to pass to temporal the cause of the error and not all the
	// stack trace.
	cause := errorsutils.Cause(err)

	switch {
	// Do not retry the following errors
	case errors.Is(err, plugins.ErrNotImplemented):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeUnimplemented, cause)
	case errors.Is(err, plugins.ErrInvalidClientRequest):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeInvalidArgument, cause)
	case errors.Is(err, plugins.ErrCurrencyNotSupported):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeInvalidArgument, cause)
	case errors.Is(err, connectors.ErrNotFound):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeInvalidArgument, cause)
	case errors.Is(err, models.ErrMissingConnectorMetadata):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeInvalidArgument, cause)
	case errors.Is(err, models.ErrWebhookVerification):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeInvalidArgument, cause)
	case errors.As(err, &models.NonRetryableError):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeInvalidArgument, cause)

	// Potentially retry
	case errors.Is(err, plugins.ErrUpstreamRatelimit):
		// Honor the upstream's wait hint when the plugin parsed one
		// from RFC 9110 Retry-After or draft-ietf-httpapi-ratelimit-headers.
		// Falls back to the static engine delay when no hint is available
		// (or when the hint is shorter than the configured floor).
		// https://docs.temporal.io/encyclopedia/retry-policies#per-error-next-retry-delay
		nextDelay := a.rateLimitingRetryDelay
		var rl *plugins.RateLimitedError
		if errors.As(err, &rl) && rl.RetryAfter > nextDelay {
			nextDelay = rl.RetryAfter
		}
		return temporal.NewApplicationErrorWithOptions(err.Error(), ErrTypeRateLimited, temporal.ApplicationErrorOptions{
			NextRetryDelay: nextDelay,
		})
	case errors.Is(err, plugins.ErrUpstreamTimeout):
		return temporal.NewApplicationErrorWithCause(err.Error(), ErrTypeTimeout, cause)
	case errors.Is(err, plugins.ErrUpstreamRetryAfter):
		return temporal.NewApplicationErrorWithOptions(err.Error(), ErrTypeRetryAfter, temporal.ApplicationErrorOptions{
			NextRetryDelay: a.rateLimitingRetryDelay,
		})

	// Retry the following errors
	case errors.Is(err, plugins.ErrNotYetInstalled):
		// We want to retry in case of not installed
		return temporal.NewApplicationErrorWithCause(err.Error(), ErrTypeDefault, cause)
	default:
		return temporal.NewApplicationErrorWithCause(err.Error(), ErrTypeDefault, cause)
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
