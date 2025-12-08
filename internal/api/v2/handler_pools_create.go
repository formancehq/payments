package v2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/common"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type CreatePoolRequest struct {
	Name       string         `json:"name" validate:"required"`
	Query      map[string]any `json:"query" validate:"required_without=AccountIDs"`
	AccountIDs []string       `json:"accountIDs" validate:"required_without=Query,dive,accountID"`
}

type PoolResponse struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Type     models.PoolType `json:"type"`
	Query    map[string]any  `json:"query,omitempty"`
	Accounts []string        `json:"accounts"`
}

func poolsCreate(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_poolsBalancesAt")
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

		if len(CreatePoolRequest.Query) > 0 {
			pool.Type = models.POOL_TYPE_DYNAMIC
			pool.Query = CreatePoolRequest.Query
		} else {
			pool.Type = models.POOL_TYPE_STATIC
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

		data := &PoolResponse{
			ID:       pool.ID.String(),
			Name:     pool.Name,
			Type:     pool.Type,
			Query:    pool.Query,
			Accounts: CreatePoolRequest.AccountIDs,
		}

		err = json.NewEncoder(w).Encode(api.BaseResponse[PoolResponse]{
			Data: data,
		})
		if err != nil {
			otel.RecordError(span, err)
			common.InternalServerError(w, r, err)
			return
		}
	}
}

func populateSpanFromCreatePoolRequest(span trace.Span, req CreatePoolRequest) {
	span.SetAttributes(attribute.String("name", req.Name))
	for i, acc := range req.AccountIDs {
		span.SetAttributes(attribute.String(fmt.Sprintf("accountIDs[%d]", i), acc))
	}
}
