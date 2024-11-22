package registry

import (
	"errors"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
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
		return fmt.Errorf("%w: %w", err, plugins.ErrInvalidClientRequest)
	default:
		return err
	}
}
