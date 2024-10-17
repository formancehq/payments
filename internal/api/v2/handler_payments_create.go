package v2

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
)

type createPaymentRequest struct {
	Reference            string            `json:"reference"`
	ConnectorID          string            `json:"connectorID"`
	CreatedAt            time.Time         `json:"createdAt"`
	Type                 string            `json:"type"`
	Amount               *big.Int          `json:"amount"`
	Asset                string            `json:"asset"`
	Scheme               string            `json:"scheme"`
	Status               string            `json:"status"`
	SourceAccountID      *string           `json:"sourceAccountID"`
	DestinationAccountID *string           `json:"destinationAccountID"`
	Metadata             map[string]string `json:"metadata"`
}

func (r *createPaymentRequest) validate() error {
	if r.Reference == "" {
		return errors.New("reference is required")
	}

	if r.ConnectorID == "" {
		return errors.New("connectorID is required")
	}

	if r.CreatedAt.IsZero() || r.CreatedAt.After(time.Now()) {
		return errors.New("createdAt is empty or in the future")
	}

	if r.Amount == nil {
		return errors.New("amount is required")
	}

	if r.Type == "" {
		return errors.New("type is required")
	}

	if _, err := models.PaymentTypeFromString(r.Type); err != nil {
		return fmt.Errorf("invalid type: %w", err)
	}

	if r.Scheme == "" {
		return errors.New("scheme is required")
	}

	if _, err := models.PaymentSchemeFromString(r.Scheme); err != nil {
		return fmt.Errorf("invalid scheme: %w", err)
	}

	if r.Asset == "" {
		return errors.New("asset is required")
	}

	if r.Status == "" {
		return errors.New("status is required")
	}

	if _, err := models.PaymentStatusFromString(r.Status); err != nil {
		return fmt.Errorf("invalid status: %w", err)
	}

	if r.SourceAccountID != nil {
		_, err := models.AccountIDFromString(*r.SourceAccountID)
		if err != nil {
			return fmt.Errorf("invalid sourceAccountID: %w", err)
		}
	}

	if r.DestinationAccountID != nil {
		_, err := models.AccountIDFromString(*r.DestinationAccountID)
		if err != nil {
			return fmt.Errorf("invalid destinationAccountID: %w", err)
		}
	}

	return nil
}

func paymentsCreate(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_paymentsCreate")
		defer span.End()

		var req createPaymentRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		if err := req.validate(); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		connectorID := models.MustConnectorIDFromString(req.ConnectorID)
		paymentType := models.MustPaymentTypeFromString(req.Type)
		status := models.MustPaymentStatusFromString(req.Status)
		raw, err := json.Marshal(req)
		if err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}
		pid := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: req.Reference,
				Type:      paymentType,
			},
			ConnectorID: connectorID,
		}

		payment := models.Payment{
			ID:            pid,
			ConnectorID:   connectorID,
			Reference:     req.Reference,
			CreatedAt:     req.CreatedAt.UTC(),
			Type:          paymentType,
			InitialAmount: req.Amount,
			Amount:        req.Amount,
			Asset:         req.Asset,
			Scheme:        models.MustPaymentSchemeFromString(req.Scheme),
			SourceAccountID: func() *models.AccountID {
				if req.SourceAccountID == nil {
					return nil
				}
				return pointer.For(models.MustAccountIDFromString(*req.SourceAccountID))
			}(),
			DestinationAccountID: func() *models.AccountID {
				if req.DestinationAccountID == nil {
					return nil
				}
				return pointer.For(models.MustAccountIDFromString(*req.DestinationAccountID))
			}(),
			Metadata: req.Metadata,
		}

		// Create adjustments from main payments to keep the compatibility with the old API
		payment.Adjustments = []models.PaymentAdjustment{
			{
				ID: models.PaymentAdjustmentID{
					PaymentID: pid,
					Reference: req.Reference,
					CreatedAt: req.CreatedAt,
					Status:    status,
				},
				PaymentID: pid,
				Reference: req.Reference,
				CreatedAt: req.CreatedAt,
				Status:    status,
				Amount:    req.Amount,
				Asset:     &req.Asset,
				Metadata:  req.Metadata,
				Raw:       raw,
			},
		}

		err = backend.PaymentsCreate(ctx, payment)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		// Compatibility with old API
		data := paymentResponse{
			ID:            payment.ID.String(),
			Reference:     payment.Reference,
			Type:          payment.Type.String(),
			Provider:      payment.ConnectorID.Provider,
			ConnectorID:   payment.ConnectorID.String(),
			Status:        payment.Status.String(),
			Amount:        payment.Amount,
			InitialAmount: payment.InitialAmount,
			Scheme:        payment.Scheme.String(),
			Asset:         payment.Asset,
			CreatedAt:     payment.CreatedAt,
			Metadata:      payment.Metadata,
		}

		if payment.SourceAccountID != nil {
			data.SourceAccountID = payment.SourceAccountID.String()
		}

		if payment.DestinationAccountID != nil {
			data.DestinationAccountID = payment.DestinationAccountID.String()
		}

		data.Adjustments = make([]paymentAdjustment, len(payment.Adjustments))
		for i := range payment.Adjustments {
			data.Adjustments[i] = paymentAdjustment{
				Reference: payment.Adjustments[i].ID.Reference,
				CreatedAt: payment.Adjustments[i].CreatedAt,
				Status:    payment.Adjustments[i].Status.String(),
				Amount:    payment.Adjustments[i].Amount,
				Raw:       payment.Adjustments[i].Raw,
			}
		}

		err = json.NewEncoder(w).Encode(api.BaseResponse[paymentResponse]{
			Data: &data,
		})
		if err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}
	}
}
