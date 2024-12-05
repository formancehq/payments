package v3

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type PaymentInitiationsCreateRequest struct {
	Reference   string    `json:"reference"`
	ScheduledAt time.Time `json:"scheduledAt"`
	ConnectorID string    `json:"connectorID"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Amount      *big.Int  `json:"amount"`
	Asset       string    `json:"asset"`

	SourceAccountID      *string `json:"sourceAccountID"`
	DestinationAccountID *string `json:"destinationAccountID"`

	Metadata map[string]string `json:"metadata"`
}

func (r *PaymentInitiationsCreateRequest) Validate() error {
	if r.Reference == "" {
		return errors.New("reference is required")
	}

	if r.SourceAccountID != nil {
		_, err := models.AccountIDFromString(*r.SourceAccountID)
		if err != nil {
			return err
		}
	}

	if r.DestinationAccountID != nil {
		_, err := models.AccountIDFromString(*r.DestinationAccountID)
		if err != nil {
			return err
		}
	}

	_, err := models.PaymentInitiationTypeFromString(r.Type)
	if err != nil {
		return err
	}

	if r.Amount == nil {
		return errors.New("amount is required")
	}

	if r.Asset == "" {
		return errors.New("asset is required")
	}

	return nil
}

type PaymentInitiationsCreateResponse struct {
	PaymentInitiationID string `json:"paymentInitiationID"`
	TaskID              string `json:"taskID"`
}

func paymentInitiationsCreate(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentInitiationsCreate")
		defer span.End()

		payload := PaymentInitiationsCreateRequest{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromPaymentInitiationCreateRequest(span, payload)

		if err := payload.Validate(); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		connectorID, err := models.ConnectorIDFromString(payload.ConnectorID)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		noValidation := r.URL.Query().Get("noValidation") == "true"

		pi := models.PaymentInitiation{
			ID: models.PaymentInitiationID{
				Reference:   payload.Reference,
				ConnectorID: connectorID,
			},
			ConnectorID: connectorID,
			Reference:   payload.Reference,
			CreatedAt:   time.Now(),
			ScheduledAt: payload.ScheduledAt,
			Description: payload.Description,
			Type:        models.MustPaymentInitiationTypeFromString(payload.Type),
			Amount:      payload.Amount,
			Asset:       payload.Asset,
			Metadata:    payload.Metadata,
		}

		if payload.SourceAccountID != nil {
			pi.SourceAccountID = pointer.For(models.MustAccountIDFromString(*payload.SourceAccountID))
		}

		if payload.DestinationAccountID != nil {
			pi.DestinationAccountID = pointer.For(models.MustAccountIDFromString(*payload.DestinationAccountID))
		}

		task, err := backend.PaymentInitiationsCreate(ctx, pi, noValidation, false)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Accepted(w, PaymentInitiationsCreateResponse{
			PaymentInitiationID: pi.ID.String(),
			TaskID:              task.ID.String(),
		})
	}
}

func populateSpanFromPaymentInitiationCreateRequest(span trace.Span, req PaymentInitiationsCreateRequest) {
	span.SetAttributes(attribute.String("reference", req.Reference))
	span.SetAttributes(attribute.String("connectorID", req.ConnectorID))
	span.SetAttributes(attribute.String("scheduledAt", req.ScheduledAt.String()))
	span.SetAttributes(attribute.String("description", req.Description))
	span.SetAttributes(attribute.String("type", req.Type))
	span.SetAttributes(attribute.String("amount", req.Amount.String()))
	span.SetAttributes(attribute.String("asset", req.Asset))
	for k, v := range req.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata[%s]", k), v))
	}
	if req.SourceAccountID != nil {
		span.SetAttributes(attribute.String("sourceAccountID", *req.SourceAccountID))
	}
	if req.DestinationAccountID != nil {
		span.SetAttributes(attribute.String("destinationAccountID", *req.DestinationAccountID))
	}
}
