package v2

import (
	"errors"
	"github.com/formancehq/go-libs/v3/platform/postgres"
	"github.com/formancehq/payments/internal/connectors/engine"
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/common"
	"github.com/formancehq/payments/internal/api/services"
	"github.com/formancehq/payments/internal/storage"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

const (
	ErrUniqueReference                 = "CONFLICT"
	ErrNotFound                        = "NOT_FOUND"
	ErrInvalidID                       = "INVALID_ID"
	ErrMissingOrInvalidBody            = "MISSING_OR_INVALID_BODY"
	ErrValidation                      = "VALIDATION"
	ErrConnectorCapabilityNotSupported = "CONNECTOR_CAPABILITY_NOT_SUPPORTED"
)

func handleServiceErrors(w http.ResponseWriter, r *http.Request, err error) {
	var capabilityNotSupported *engine.ErrConnectorCapabilityNotSupported

	switch {
	case errors.Is(err, storage.ErrDuplicateKeyValue), postgres.ErrConstraintsFailed{}.Is(err):
		api.BadRequest(w, ErrUniqueReference, err)
	case errors.Is(err, storage.ErrNotFound), errors.Is(err, postgres.ErrNotFound):
		api.NotFound(w, err)
	case errors.Is(err, storage.ErrValidation):
		api.BadRequest(w, ErrValidation, err)
	case errors.Is(err, services.ErrValidation), postgres.ErrValidationFailed{}.Is(err):
		cause := errorsutils.Cause(err)
		api.BadRequest(w, ErrValidation, cause)
	case errors.Is(err, services.ErrNotFound):
		api.NotFound(w, err)
	case errors.As(err, &capabilityNotSupported):
		api.BadRequest(w, ErrConnectorCapabilityNotSupported, err)
	default:
		common.InternalServerError(w, r, err)
	}
}
