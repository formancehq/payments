package engine

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/temporal"
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("not found")
)

// handleWorkflowError processes Temporal workflow errors and wraps validation errors
// with ErrValidation to provide consistent error handling for API responses.
func handleWorkflowError(err error) error {
	var applicationErr *temporal.ApplicationError
	if errors.As(err, &applicationErr) {
		switch applicationErr.Type() {
		case activities.ErrTypeInvalidArgument:
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
