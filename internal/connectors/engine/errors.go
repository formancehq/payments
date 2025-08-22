package engine

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/temporal"
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("not found")
)

type ErrConnectorCapabilityNotSupported struct {
	Capability string
	Provider   string
}

func (e *ErrConnectorCapabilityNotSupported) Error() string {
	return fmt.Sprintf("%s capability is not supported by the provider %s. Check here the supported features: https://docs.formance.com/payments/connectors/#supported-processors", e.Capability, e.Provider)
}

// handleWorkflowError processes Temporal workflow errors and wraps validation errors
// with ErrValidation to provide consistent error handling for API responses.
func handleWorkflowError(err error) error {
	var applicationErr *temporal.ApplicationError
	if errors.As(err, &applicationErr) {
		switch applicationErr.Type() {
		case activities.ErrTypeInvalidArgument, workflow.ErrValidation:
			return errorsutils.NewWrappedError(
				errorsutils.Cause(err),
				ErrValidation,
			)
		default:
			return err
		}
	}

	return err
}

func handlePluginErrors(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, models.ErrInvalidRequest), errors.Is(err, models.ErrInvalidConfig):
		return errorsutils.NewWrappedError(err, ErrValidation)
	default:
		return err
	}
}
