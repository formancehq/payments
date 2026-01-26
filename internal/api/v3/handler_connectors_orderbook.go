package v3

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

func connectorsOrderBook(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_connectorsOrderBook")
		defer span.End()

		span.SetAttributes(attribute.String("connectorID", connectorID(r)))
		connectorID, err := models.ConnectorIDFromString(connectorID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		pair := r.URL.Query().Get("pair")
		if pair == "" {
			api.BadRequest(w, ErrValidation, fmt.Errorf("pair query parameter is required"))
			return
		}

		depth := 0
		if depthStr := r.URL.Query().Get("depth"); depthStr != "" {
			depth, err = strconv.Atoi(depthStr)
			if err != nil || depth < 0 {
				api.BadRequest(w, ErrValidation, fmt.Errorf("depth must be a non-negative integer"))
				return
			}
		}

		orderBook, err := backend.ConnectorsGetOrderBook(ctx, connectorID, pair, depth)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Ok(w, orderBook)
	}
}
