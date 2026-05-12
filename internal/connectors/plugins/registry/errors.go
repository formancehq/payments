package registry

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func translateError(err error) error {
	switch {
	case errors.Is(err, plugins.ErrNotImplemented):
		return err
	// Already a typed RateLimitedError (httpwrapper default-on path):
	// preserve it as-is so the engine's temporalPluginErrorCheck can
	// extract RetryAfter via errors.As. Skipping the wrap also avoids
	// a redundant ErrUpstreamRatelimit in the error chain.
	case isRateLimited(err):
		return err
	case errors.Is(err, models.ErrMissingFromPayloadInRequest),
		errors.Is(err, models.ErrMissingAccountInRequest),
		errors.Is(err, models.ErrInvalidRequest),
		errors.Is(err, plugins.ErrCurrencyNotSupported),
		errors.Is(err, httpwrapper.ErrStatusCodeClientError),
		errors.Is(err, models.ErrInvalidConfig):
		return errorsutils.NewWrappedError(
			err,
			plugins.ErrInvalidClientRequest,
		)
	case errors.Is(err, httpwrapper.ErrStatusCodeTooManyRequests):
		return errorsutils.NewWrappedError(
			err,
			plugins.ErrUpstreamRatelimit,
		)
	case errors.Is(err, httpwrapper.ErrStatusCodeRequestTimeout),
		errors.Is(err, httpwrapper.ErrStatusCodeMisdirectedRequest):
		return errorsutils.NewWrappedError(
			err,
			plugins.ErrUpstreamTimeout,
		)
	case errors.Is(err, httpwrapper.ErrStatusCodeLocked),
		errors.Is(err, httpwrapper.ErrStatusCodeTooEarly):
		return errorsutils.NewWrappedError(
			err,
			plugins.ErrUpstreamRetryAfter,
		)
	default:
		return err
	}
}

func isRateLimited(err error) bool {
	var rl *plugins.RateLimitedError
	return errors.As(err, &rl)
}
