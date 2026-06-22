package registry

import (
	"errors"

	"github.com/formancehq/payments/pkg/domain/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/pkg/domain/models"
	errorsutils "github.com/formancehq/payments/pkg/domain/errors"
)

func translateError(err error) error {
	switch {
	case errors.Is(err, plugins.ErrNotImplemented):
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
