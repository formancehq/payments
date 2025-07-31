package v3

import (
	"fmt"
	"github.com/formancehq/go-libs/v3/platform/postgres"
	"net/http"
	"strings"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/common"
	"github.com/formancehq/payments/internal/api/services"
	"github.com/formancehq/payments/internal/connectors/engine"
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

var FKViolationColumn = []string{
	"connector_id",
	"bank_account_id",
	"payment_id",
	"pool_id",
	"schedule_id",
	"payment_initiation_id",
	"payment_initiation_reversal_id",
}

func handleServiceErrors(w http.ResponseWriter, r *http.Request, err error) {
	var capabilityNotSupported *engine.ErrConnectorCapabilityNotSupported
	var errFKConstraintFailed postgres.ErrFKConstraintFailed

	switch {
	case errors.Is(err, storage.ErrDuplicateKeyValue), postgres.ErrConstraintsFailed{}.Is(err):
		api.BadRequest(w, ErrUniqueReference, err)
	case errors.Is(err, storage.ErrNotFound), errors.Is(err, postgres.ErrNotFound):
		api.NotFound(w, err)
	case errors.Is(err, storage.ErrForeignKeyViolation), postgres.ErrFKConstraintFailed{}.Is(err):
		if errors.As(err, &errFKConstraintFailed) {
			for _, column := range FKViolationColumn {
				if strings.Contains(errFKConstraintFailed.GetConstraint(), column) {
					err = fmt.Errorf("%s: %w", column, storage.ErrForeignKeyViolation)
				}
			}
		}

		api.BadRequest(w, ErrValidation, errors.Cause(err))
	case errors.Is(err, storage.ErrValidation), postgres.ErrValidationFailed{}.Is(err):
		api.BadRequest(w, ErrValidation, err)
	case errors.Is(err, services.ErrValidation):
		api.BadRequest(w, ErrValidation, err)
	case errors.Is(err, services.ErrNotFound):
		api.NotFound(w, err)
	case errors.As(err, &capabilityNotSupported):
		api.BadRequest(w, ErrConnectorCapabilityNotSupported, err)
	default:
		common.InternalServerError(w, r, err)
	}
}
