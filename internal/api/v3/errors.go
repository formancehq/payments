package v3

import (
	"errors"
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/common"
	"github.com/formancehq/payments/internal/api/services"
	"github.com/formancehq/payments/internal/storage"
)

const (
	ErrValidation           = "VALIDATION"
	ErrInvalidID            = "INVALID_ID"
	ErrMissingOrInvalidBody = "MISSING_OR_INVALID_BODY"
	ErrUniqueReference      = "CONFLICT"
)

func handleServiceErrors(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, storage.ErrDuplicateKeyValue):
		api.BadRequest(w, ErrUniqueReference, err)
	case errors.Is(err, storage.ErrNotFound):
		api.NotFound(w, err)
	case errors.Is(err, storage.ErrValidation):
		api.BadRequest(w, ErrValidation, err)
	case errors.Is(err, services.ErrValidation):
		api.BadRequest(w, ErrValidation, err)
	case errors.Is(err, services.ErrNotFound):
		api.NotFound(w, err)
	default:
		common.InternalServerError(w, r, err)
	}
}
