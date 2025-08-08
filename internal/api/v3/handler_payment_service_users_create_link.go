package v3

import (
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

type PaymentServiceUserCreateLinkRequest struct {
	ApplicationName   string `json:"applicationName" validate:"required"`
	ClientRedirectURL string `json:"clientRedirectURL" validate:"required,url"`
}

type PaymentServiceUserCreateLinkResponse struct {
	AttemptID string `json:"attemptID"`
	Link      string `json:"link"`
}

func paymentServiceUsersCreateLink(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentServiceUsersCreateLink")
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

		queryValues := r.URL.Query()
		var ik *uuid.UUID
		idempotencyKey, ok := queryValues["Idempotency-Key"]
		if !ok || len(idempotencyKey) == 0 || idempotencyKey[0] == "" {
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

		if r.Body == nil {
			otel.RecordError(span, fmt.Errorf("body is nil"))
			api.BadRequest(w, ErrMissingOrInvalidBody, fmt.Errorf("body is nil"))
			return
		}

		var req PaymentServiceUserCreateLinkRequest
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

		attemptID, link, err := backend.PaymentServiceUsersCreateLink(ctx, req.ApplicationName, id, connectorID, ik, &req.ClientRedirectURL)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		// Since we send a link to the client, we need to disable HTML escaping
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := encoder.Encode(PaymentServiceUserCreateLinkResponse{
			AttemptID: attemptID,
			Link:      link,
		}); err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}
	}
}
