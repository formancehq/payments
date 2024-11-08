package v3

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type createPoolRequest struct {
	Name       string   `json:"name"`
	AccountIDs []string `json:"accountIDs"`
}

func (r *createPoolRequest) Validate() error {
	if len(r.AccountIDs) == 0 {
		return errors.New("one or more account id required")
	}
	return nil
}

func poolsCreate(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_poolsCreate")
		defer span.End()

		var createPoolRequest createPoolRequest
		err := json.NewDecoder(r.Body).Decode(&createPoolRequest)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromCreatePoolRequest(span, createPoolRequest)

		if err := createPoolRequest.Validate(); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		pool := models.Pool{
			ID:        uuid.New(),
			Name:      createPoolRequest.Name,
			CreatedAt: time.Now().UTC(),
		}

		accounts := make([]models.PoolAccounts, len(createPoolRequest.AccountIDs))
		for i, accountID := range createPoolRequest.AccountIDs {
			aID, err := models.AccountIDFromString(accountID)
			if err != nil {
				otel.RecordError(span, err)
				api.BadRequest(w, ErrValidation, err)
				return
			}

			accounts[i] = models.PoolAccounts{
				PoolID:    pool.ID,
				AccountID: aID,
			}
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

func populateSpanFromCreatePoolRequest(span trace.Span, req createPoolRequest) {
	span.SetAttributes(attribute.String("name", req.Name))
	for i, acc := range req.AccountIDs {
		span.SetAttributes(attribute.String(fmt.Sprintf("accountIDs[%d]", i), acc))
	}
}
