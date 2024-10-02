package activities

import (
	"errors"

	engineplugins "github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/temporal"
)

var nonRetryableErrors = []error{
	engineplugins.ErrNotFound,
	httpwrapper.ErrStatusCodeClientError,
	models.ErrMissingFromPayloadInRequest,
	models.ErrMissingAccountInMetadata,
	plugins.ErrNotYetInstalled,
	plugins.ErrNotImplemented,
}

func temporalError(err error, cause string) error {
	isRetryable := true

	for _, candidate := range nonRetryableErrors {
		if errors.Is(err, candidate) {
			isRetryable = false
			break
		}
	}

	if isRetryable {
		return temporal.NewApplicationErrorWithCause(err.Error(), cause, err)
	}
	return temporal.NewNonRetryableApplicationError(err.Error(), cause, err)
}
