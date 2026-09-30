package engine

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/pkg/domain/models"
	errorsutils "github.com/formancehq/payments/pkg/domain/errors"
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
	return fmt.Sprintf("%s capability is not supported by the provider %s. Check here the supported features: https://docs.formance.com/modules/connectivity/capabilities", e.Capability, e.Provider)
}

// notFoundError reports a missing object while keeping the underlying message
// as-is: it already spells out what was not found, so there is nothing to add.
type notFoundError struct {
	msg string
}

func (e *notFoundError) Error() string { return e.msg }

func (e *notFoundError) Unwrap() error { return ErrNotFound }

// handleWorkflowError processes Temporal workflow errors and wraps validation
// and not found errors with ErrValidation/ErrNotFound to provide consistent
// error handling for API responses.
func handleWorkflowError(err error) error {
	var applicationErr *temporal.ApplicationError
	if errors.As(err, &applicationErr) {
		switch applicationErr.Type() {
		case activities.ErrTypeInvalidArgument, workflow.ErrValidation:
			return errorsutils.NewWrappedError(
				errorsutils.Cause(err),
				ErrValidation,
			)
		case activities.ErrTypeStorageNotFound:
			// Message() is the only clean rendering of the failure: the error
			// itself is decorated by temporal with its type and retry policy.
			return &notFoundError{msg: applicationErr.Message()}
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
