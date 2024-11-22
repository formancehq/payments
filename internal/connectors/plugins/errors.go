package plugins

import (
	"errors"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/models"
)

var (
	ErrNotImplemented       = errors.New("not implemented")
	ErrNotYetInstalled      = errors.New("not yet installed")
	ErrInvalidClientRequest = errors.New("invalid client request")
	ErrCurrencyNotSupported = errors.New("currency not supported")
)

func translateError(err error) error {
	switch {
	case errors.Is(err, ErrNotImplemented):
		return err
	case errors.Is(err, models.ErrMissingFromPayloadInRequest),
		errors.Is(err, models.ErrMissingAccountInRequest),
		errors.Is(err, models.ErrInvalidRequest),
		errors.Is(err, ErrCurrencyNotSupported),
		errors.Is(err, httpwrapper.ErrStatusCodeClientError),
		errors.Is(err, models.ErrInvalidConfig):
		return fmt.Errorf("%w: %w", err, ErrInvalidClientRequest)
	default:
		return err
	}
}
