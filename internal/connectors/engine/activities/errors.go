package activities

import (
	"errors"

	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/temporal"
)

var nonRetryableErrors = []error{
	httpwrapper.ErrStatusCodeClientError,
	models.ErrMissingFromPayloadInRequest,
	models.ErrMissingAccountInMetadata,
	plugins.ErrNotFound,
}

func temporalError(err error) error {
	isRetryable := true

	for _, candidate := range nonRetryableErrors {
		if errors.Is(err, candidate) {
			isRetryable = false
			break
		}
	}

	if isRetryable {
		return temporal.NewApplicationErrorWithCause(err.Error(), "application", err)
	}
	return temporal.NewNonRetryableApplicationError(err.Error(), "application", err)
}
