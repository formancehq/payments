package engine

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/temporal"
)

var (
	ErrValidation                    = errors.New("validation error")
	ErrNotFound                      = errors.New("not found")
	ErrConnectorCapacityNotSupported = errors.New("this connector does not support this capacity")
)

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
