package v3

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

func connectorsOHLC(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_connectorsOHLC")
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

		interval := r.URL.Query().Get("interval")
		if interval == "" {
			interval = "1h" // Default to 1 hour
		}

		// Validate interval
		validIntervals := map[string]bool{
			"1m": true, "5m": true, "15m": true, "30m": true,
			"1h": true, "4h": true, "1d": true, "1w": true,
		}
		if !validIntervals[interval] {
			api.BadRequest(w, ErrValidation, fmt.Errorf("invalid interval: %s. Valid intervals: 1m, 5m, 15m, 30m, 1h, 4h, 1d, 1w", interval))
			return
		}

		req := models.GetOHLCRequest{
			Pair:     pair,
			Interval: interval,
		}

		// Parse optional since parameter
		if since := r.URL.Query().Get("since"); since != "" {
			t, err := time.Parse(time.RFC3339, since)
			if err != nil {
				api.BadRequest(w, ErrValidation, fmt.Errorf("invalid since format, use RFC3339: %w", err))
				return
			}
			req.Since = &t
		}

		// Parse optional limit parameter
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				api.BadRequest(w, ErrValidation, fmt.Errorf("invalid limit: %w", err))
				return
			}
			req.Limit = limit
		}

		ohlc, err := backend.ConnectorsGetOHLC(ctx, connectorID, req)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Ok(w, ohlc)
	}
}
