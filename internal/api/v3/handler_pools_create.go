package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type CreatePoolRequest struct {
	Name       string   `json:"name" validate:"required"`
	AccountIDs []string `json:"accountIDs" validate:"min=1,dive,accountID"`
}

func poolsCreate(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_poolsCreate")
		defer span.End()

		var CreatePoolRequest CreatePoolRequest
		err := json.NewDecoder(r.Body).Decode(&CreatePoolRequest)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromCreatePoolRequest(span, CreatePoolRequest)

		if _, err := validator.Validate(CreatePoolRequest); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		pool := models.Pool{
			ID:        uuid.New(),
			Name:      CreatePoolRequest.Name,
			CreatedAt: time.Now().UTC(),
		}

		accounts := make([]models.AccountID, len(CreatePoolRequest.AccountIDs))
		for i, accountID := range CreatePoolRequest.AccountIDs {
			aID, err := models.AccountIDFromString(accountID)
			if err != nil {
				otel.RecordError(span, err)
				api.BadRequest(w, ErrValidation, err)
				return
			}

			accounts[i] = aID
		}
		pool.PoolAccounts = accounts

		err = backend.PoolsCreate(ctx, pool)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Created(w, pool.ID.String())
	}
}

func populateSpanFromCreatePoolRequest(span trace.Span, req CreatePoolRequest) {
	span.SetAttributes(attribute.String("name", req.Name))
	for i, acc := range req.AccountIDs {
		span.SetAttributes(attribute.String(fmt.Sprintf("accountIDs[%d]", i), acc))
	}
}
