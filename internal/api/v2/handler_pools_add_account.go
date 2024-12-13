package v2

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type PoolsAddAccountRequest struct {
	AccountID string `json:"accountID"`
}

func (c *PoolsAddAccountRequest) Validate() error {
	if c.AccountID == "" {
		return errors.New("accountID is required")
	}

	return nil
}

func poolsAddAccount(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_poolsAddAccount")
		defer span.End()

		span.SetAttributes(attribute.String("poolID", poolID(r)))
		id, err := uuid.Parse(poolID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		var PoolsAddAccountRequest PoolsAddAccountRequest
		err = json.NewDecoder(r.Body).Decode(&PoolsAddAccountRequest)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		span.SetAttributes(attribute.String("accountID", PoolsAddAccountRequest.AccountID))

		if err := PoolsAddAccountRequest.Validate(); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		accountID, err := models.AccountIDFromString(PoolsAddAccountRequest.AccountID)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		err = backend.PoolsAddAccount(ctx, id, accountID)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.NoContent(w)
	}
}
