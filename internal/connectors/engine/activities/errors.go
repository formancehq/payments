package activities

import (
	"context"
	"errors"

	enginePlugins "github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

const (
	ErrTypeStorage         = "STORAGE"
	ErrTypeDefault         = "DEFAULT"
	ErrTypeInvalidArgument = "INVALID_ARGUMENT"
	ErrTypeRateLimited     = "RATE_LIMITED"
	ErrTypeUnimplemented   = "UNIMPLEMENTED"
)

func (a Activities) temporalPluginError(ctx context.Context, err error) error {
	return a.temporalPluginErrorCheck(ctx, err, false)
}

func (a Activities) temporalPluginPollingError(ctx context.Context, err error, periodic bool) error {
	return a.temporalPluginErrorCheck(ctx, err, periodic)
}

func (a Activities) temporalPluginErrorCheck(ctx context.Context, err error, isPeriodic bool) error {
	switch {
	// Do not retry the following errors
	case errors.Is(err, plugins.ErrNotImplemented):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeUnimplemented, err)
	case errors.Is(err, plugins.ErrInvalidClientRequest):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeInvalidArgument, err)
	case errors.Is(err, plugins.ErrCurrencyNotSupported):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeInvalidArgument, err)
	case errors.Is(err, enginePlugins.ErrNotFound):
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeInvalidArgument, err)

	// Potentially retry
	case errors.Is(err, plugins.ErrUpstreamRatelimit):
		// periodic tasks will be repeated in the future anyway so we can skip retry in case of rate-limiting
		if isPeriodic {
			info := activity.GetInfo(ctx)
			a.logger.WithFields(map[string]any{
				"workflow_type":  info.WorkflowType.Name,
				"scheduled_time": info.ScheduledTime.String(),
				"workflow_id":    info.WorkflowExecution.ID,
			}).Debug("disabling retry for polled activity triggered by schedule due to rate-limit")
			return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeRateLimited, err)
		}

		return temporal.NewApplicationErrorWithOptions(err.Error(), ErrTypeRateLimited, temporal.ApplicationErrorOptions{
			// temporal already implements a backoff strategy, but let's add an extra delay before the next retry
			// https://docs.temporal.io/encyclopedia/retry-policies#per-error-next-retry-delay
			NextRetryDelay: a.rateLimitingRetryDelay,
		})

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
