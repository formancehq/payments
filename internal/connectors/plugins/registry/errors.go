package registry

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/formancehq/payments/pkg/connector"
)

func translateError(err error) error {
	switch {
	case errors.Is(err, connector.ErrNotImplemented):
		return err
	case errors.Is(err, models.ErrMissingFromPayloadInRequest),
		errors.Is(err, models.ErrMissingAccountInRequest),
		errors.Is(err, models.ErrInvalidRequest),
		errors.Is(err, connector.ErrCurrencyNotSupported),
		errors.Is(err, httpwrapper.ErrStatusCodeClientError),
		errors.Is(err, models.ErrInvalidConfig):
		return errorsutils.NewWrappedError(
			err,
			connector.ErrInvalidClientRequest,
		)
	case errors.Is(err, httpwrapper.ErrStatusCodeTooManyRequests):
		return errorsutils.NewWrappedError(
			err,
			connector.ErrUpstreamRatelimit,
		)
	default:
		return err
	}
}
