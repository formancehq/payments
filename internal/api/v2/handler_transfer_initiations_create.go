package v2

import (
	"encoding/json"
	"errors"
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

type createTransferInitiationRequest struct {
	Reference            string            `json:"reference"`
	ScheduledAt          time.Time         `json:"scheduledAt"`
	Description          string            `json:"description"`
	SourceAccountID      string            `json:"sourceAccountID"`
	DestinationAccountID string            `json:"destinationAccountID"`
	ConnectorID          string            `json:"connectorID"`
	Provider             string            `json:"provider"`
	Type                 string            `json:"type"`
	Amount               *big.Int          `json:"amount"`
	Asset                string            `json:"asset"`
	Validated            bool              `json:"validated"`
	Metadata             map[string]string `json:"metadata"`
}

func (r *createTransferInitiationRequest) Validate() error {
	if r.Reference == "" {
		return errors.New("reference is required")
	}

	if r.SourceAccountID != "" {
		_, err := models.AccountIDFromString(r.SourceAccountID)
		if err != nil {
			return err
		}
	}

	if r.DestinationAccountID != "" {
		_, err := models.AccountIDFromString(r.DestinationAccountID)
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

func transferInitiationsCreate(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_transferInitiationsCreate")
		defer span.End()

		payload := createTransferInitiationRequest{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		setSpanAttributesFromRequest(span, payload)

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

		if payload.SourceAccountID != "" {
			pi.SourceAccountID = pointer.For(models.MustAccountIDFromString(payload.SourceAccountID))
		}

		if payload.DestinationAccountID != "" {
			pi.DestinationAccountID = pointer.For(models.MustAccountIDFromString(payload.DestinationAccountID))
		}

		_, err = backend.PaymentInitiationsCreate(ctx, pi, payload.Validated, true)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		resp := translatePaymentInitiationToResponse(&pi)
		lastAdjustment, err := backend.PaymentInitiationAdjustmentsGetLast(ctx, pi.ID)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		if lastAdjustment != nil {
			resp.Status = lastAdjustment.Status.String()
			resp.Error = func() string {
				if lastAdjustment.Error == nil {
					return ""
				}
				return lastAdjustment.Error.Error()
			}()
		}

		err = json.NewEncoder(w).Encode(api.BaseResponse[transferInitiationResponse]{
			Data: &resp,
		})
		if err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}
	}
}

func setSpanAttributesFromRequest(span trace.Span, transfer createTransferInitiationRequest) {
	span.SetAttributes(
		attribute.String("request.reference", transfer.Reference),
		attribute.String("request.scheduledAt", transfer.ScheduledAt.String()),
		attribute.String("request.description", transfer.Description),
		attribute.String("request.sourceAccountID", transfer.SourceAccountID),
		attribute.String("request.destinationAccountID", transfer.DestinationAccountID),
		attribute.String("request.connectorID", transfer.ConnectorID),
		attribute.String("request.provider", transfer.Provider),
		attribute.String("request.type", transfer.Type),
		attribute.String("request.amount", transfer.Amount.String()),
		attribute.String("request.asset", transfer.Asset),
		attribute.String("request.validated", transfer.Asset),
	)
}
