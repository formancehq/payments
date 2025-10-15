package v3

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type PaymentServiceUserUpdateLinkRequest struct {
	ApplicationName   string `json:"applicationName"` // Note: might be mandatory for some open banking providers
	ClientRedirectURL string `json:"clientRedirectURL" validate:"required,url"`
}

type PaymentServiceUserUpdateLinkResponse struct {
	AttemptID string `json:"attemptID"`
	Link      string `json:"link"`
}

func paymentServiceUsersUpdateLink(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentServiceUsersUpdateLink")
		defer span.End()

		span.SetAttributes(attribute.String("paymentServiceUserID", paymentServiceUserID(r)))
		id, err := uuid.Parse(paymentServiceUserID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		span.SetAttributes(attribute.String("connectorID", connectorID(r)))
		connectorID, err := models.ConnectorIDFromString(connectorID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		connectionID := connectionID(r)
		span.SetAttributes(attribute.String("connectionID", connectionID))

		queryValues := r.URL.Query()
		var ik *uuid.UUID
		idempotencyKey, ok := queryValues["Idempotency-Key"]
		if !ok || len(idempotencyKey) == 0 {
			ik = nil
		} else {
			u, err := uuid.Parse(idempotencyKey[0])
			if err != nil {
				err = fmt.Errorf("parsing idempotency key (need uuid): %w", err)
				otel.RecordError(span, err)
				api.BadRequest(w, ErrInvalidID, err)
				return
			}
			ik = &u
		}

		var req PaymentServiceUserUpdateLinkRequest
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		span.SetAttributes(attribute.String("clientRedirectURL", req.ClientRedirectURL))

		_, err = validator.Validate(req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		attemptID, link, err := backend.PaymentServiceUsersUpdateLink(ctx, req.ApplicationName, id, connectorID, connectionID, ik, &req.ClientRedirectURL)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		// Since we send a link to the client, we need to disable HTML escaping
		// Encode to a buffer first to avoid sending 201 if encoding fails
		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(PaymentServiceUserUpdateLinkResponse{
			AttemptID: attemptID,
			Link:      link,
		}); err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// Optional hardening: avoid caching of link payloads
		// w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write(buf.Bytes()); err != nil {
			// Headers already sent; best effort logging only.
			otel.RecordError(span, err)
			return
		}
	}
}
