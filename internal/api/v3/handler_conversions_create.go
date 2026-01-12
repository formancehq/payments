package v3

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type CreateConversionRequest struct {
	Reference    string            `json:"reference" validate:"required,gte=3,lte=1000"`
	ConnectorID  string            `json:"connectorID" validate:"required,connectorID"`
	SourceAsset  string            `json:"sourceAsset" validate:"required,asset"`
	TargetAsset  string            `json:"targetAsset" validate:"required,asset"`
	SourceAmount *big.Int          `json:"sourceAmount" validate:"required"`
	WalletID     string            `json:"walletId" validate:"required"`
	Metadata     map[string]string `json:"metadata"`
}

func conversionsCreate(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_conversionsCreate")
		defer span.End()

		var req CreateConversionRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromConversionCreateRequest(span, req)

		if _, err := validator.Validate(req); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		connectorID := models.MustConnectorIDFromString(req.ConnectorID)
		now := time.Now().UTC()

		conversion := models.Conversion{
			ID: models.ConversionID{
				Reference:   req.Reference,
				ConnectorID: connectorID,
			},
			ConnectorID:  connectorID,
			Reference:    req.Reference,
			CreatedAt:    now,
			UpdatedAt:    now,
			SourceAsset:  req.SourceAsset,
			TargetAsset:  req.TargetAsset,
			SourceAmount: req.SourceAmount,
			Status:       models.CONVERSION_STATUS_PENDING,
			WalletID:     req.WalletID,
			Metadata:     req.Metadata,
		}

		err = backend.ConversionsCreate(ctx, conversion)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Created(w, conversion)
	}
}

func populateSpanFromConversionCreateRequest(span trace.Span, req CreateConversionRequest) {
	span.SetAttributes(attribute.String("reference", req.Reference))
	span.SetAttributes(attribute.String("connectorID", req.ConnectorID))
	span.SetAttributes(attribute.String("sourceAsset", req.SourceAsset))
	span.SetAttributes(attribute.String("targetAsset", req.TargetAsset))
	if req.SourceAmount != nil {
		span.SetAttributes(attribute.String("sourceAmount", req.SourceAmount.String()))
	}
	span.SetAttributes(attribute.String("walletId", req.WalletID))
	for k, v := range req.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata[%s]", k), v))
	}
}
