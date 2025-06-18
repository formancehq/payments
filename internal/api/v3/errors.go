package v3

import (
	"github.com/formancehq/payments/internal/connectors/engine"
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/common"
	"github.com/formancehq/payments/internal/api/services"
	"github.com/formancehq/payments/internal/storage"
	"github.com/pkg/errors"
)

const (
	ErrValidation                      = "VALIDATION"
	ErrInvalidID                       = "INVALID_ID"
	ErrMissingOrInvalidBody            = "MISSING_OR_INVALID_BODY"
	ErrUniqueReference                 = "CONFLICT"
	ErrConnectorCapabilityNotSupported = "CONNECTOR_CAPABILITY_NOT_SUPPORTED"
)

func handleServiceErrors(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, storage.ErrDuplicateKeyValue):
		api.BadRequest(w, ErrUniqueReference, err)
	case errors.Is(err, storage.ErrNotFound):
		api.NotFound(w, err)
	case errors.Is(err, storage.ErrForeignKeyViolation):
		api.BadRequest(w, ErrValidation, errors.Cause(err))
	case errors.Is(err, storage.ErrValidation):
		api.BadRequest(w, ErrValidation, err)
	case errors.Is(err, services.ErrValidation):
		api.BadRequest(w, ErrValidation, err)
	case errors.Is(err, services.ErrNotFound):
		api.NotFound(w, err)
	case errors.Is(err, &engine.ErrConnectorCapabilityNotSupported{}):
		api.BadRequest(w, ErrConnectorCapabilityNotSupported, err)
	default:
		common.InternalServerError(w, r, err)
	}
}
