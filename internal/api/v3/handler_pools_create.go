package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/api"
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
	AccountIDs []string `json:"accountIDs" validate:"omitempty,min=1,dive,accountID"`
	Query      *string  `json:"query"`
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

		if CreatePoolRequest.Query == nil && len(CreatePoolRequest.AccountIDs) == 0 {
			api.BadRequest(w, ErrValidation, fmt.Errorf("either accountIDs or query must be provided"))
			return
		}
		if CreatePoolRequest.Query != nil && len(CreatePoolRequest.AccountIDs) > 0 {
			api.BadRequest(w, ErrValidation, fmt.Errorf("accountIDs and query are mutually exclusive"))
			return
		}

		pool := models.Pool{
			ID:        uuid.New(),
			Name:      CreatePoolRequest.Name,
			CreatedAt: time.Now().UTC(),
		}

		if CreatePoolRequest.Query != nil {
			pool.Query = CreatePoolRequest.Query
		} else {
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
		}

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
	if req.Query != nil {
		span.SetAttributes(attribute.String("query", *req.Query))
	}
}
