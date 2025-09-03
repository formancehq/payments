package v3

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// Custom response type for the attempt because we don't want to expose the
// temporary token and the state. We still need them in the models object because
// of temporal, that's why we need to create a custom response type.
type attemptResponse struct {
	// ID of the attempt
	ID uuid.UUID `json:"id"`
	// ID of the psu
	PsuID uuid.UUID `json:"psuID"`
	// Related connector ID
	ConnectorID models.ConnectorID `json:"connectorID"`
	// Creation date of the attempt
	CreatedAt time.Time `json:"createdAt"`
	// Status of the attempt
	Status models.PSUOpenBankingConnectionAttemptStatus `json:"status"`
	// Client redirect URL, given by the user
	ClientRedirectURL *string `json:"clientRedirectURL"`

	// Optional
	// Error message in case of failure
	Error *string `json:"error"`
}

func paymentServiceUsersLinkAttemptGet(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentServiceUsersLinkAttemptGet")
		defer span.End()

		span.SetAttributes(attribute.String("paymentServiceUserID", paymentServiceUserID(r)))
		psuID, err := uuid.Parse(paymentServiceUserID(r))
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

		span.SetAttributes(attribute.String("attemptID", attemptID(r)))
		attemptID, err := uuid.Parse(attemptID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		attempt, err := backend.PaymentServiceUsersLinkAttemptsGet(ctx, psuID, connectorID, attemptID)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		attemptResponse := attemptResponse{
			ID:                attempt.ID,
			PsuID:             attempt.PsuID,
			ConnectorID:       attempt.ConnectorID,
			CreatedAt:         attempt.CreatedAt,
			Status:            attempt.Status,
			ClientRedirectURL: attempt.ClientRedirectURL,
			Error:             attempt.Error,
		}

		api.Ok(w, attemptResponse)
	}
}

func (a attemptResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                string    `json:"id"`
		PsuID             string    `json:"psuID"`
		ConnectorID       string    `json:"connectorID"`
		CreatedAt         time.Time `json:"createdAt"`
		Status            string    `json:"status"`
		ClientRedirectURL *string   `json:"clientRedirectURL"`
		Error             *string   `json:"error"`
	}{
		ID:                a.ID.String(),
		PsuID:             a.PsuID.String(),
		ConnectorID:       a.ConnectorID.String(),
		CreatedAt:         a.CreatedAt,
		Status:            string(a.Status),
		ClientRedirectURL: a.ClientRedirectURL,
		Error:             a.Error,
	})
}
