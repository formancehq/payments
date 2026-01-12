package v3

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

type quoteRequest struct {
	SourceAsset string `json:"sourceAsset"`
	TargetAsset string `json:"targetAsset"`
	Direction   string `json:"direction"` // BUY or SELL
	Quantity    string `json:"quantity"`
}

func connectorsQuote(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_connectorsQuote")
		defer span.End()

		span.SetAttributes(attribute.String("connectorID", connectorID(r)))
		connectorID, err := models.ConnectorIDFromString(connectorID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		var req quoteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		if req.SourceAsset == "" || req.TargetAsset == "" {
			api.BadRequest(w, ErrValidation, fmt.Errorf("sourceAsset and targetAsset are required"))
			return
		}

		if req.Direction != "BUY" && req.Direction != "SELL" {
			api.BadRequest(w, ErrValidation, fmt.Errorf("direction must be BUY or SELL"))
			return
		}

		quantity := new(big.Int)
		if req.Quantity != "" {
			_, ok := quantity.SetString(req.Quantity, 10)
			if !ok {
				api.BadRequest(w, ErrValidation, fmt.Errorf("quantity must be a valid integer"))
				return
			}
		}

		quoteReq := models.GetQuoteRequest{
			SourceAsset: req.SourceAsset,
			TargetAsset: req.TargetAsset,
			Direction:   req.Direction,
			Quantity:    quantity,
		}

		quote, err := backend.ConnectorsGetQuote(ctx, connectorID, quoteReq)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Ok(w, quote)
	}
}
