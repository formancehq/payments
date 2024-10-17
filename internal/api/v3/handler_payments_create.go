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
)

type createPaymentRequest struct {
	Reference            string                             `json:"reference"`
	ConnectorID          string                             `json:"connectorID"`
	CreatedAt            time.Time                          `json:"createdAt"`
	Type                 string                             `json:"type"`
	InitialAmount        *big.Int                           `json:"initialAmount"`
	Amount               *big.Int                           `json:"amount"`
	Asset                string                             `json:"asset"`
	Scheme               string                             `json:"scheme"`
	SourceAccountID      *string                            `json:"sourceAccountID"`
	DestinationAccountID *string                            `json:"destinationAccountID"`
	Metadata             map[string]string                  `json:"metadata"`
	Adjustments          []createPaymentsAdjustmentsRequest `json:"adjustments"`
}

type createPaymentsAdjustmentsRequest struct {
	Reference string            `json:"reference"`
	CreatedAt time.Time         `json:"createdAt"`
	Status    string            `json:"status"`
	Amount    *big.Int          `json:"amount"`
	Asset     *string           `json:"asset"`
	Metadata  map[string]string `json:"metadata"`
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

	if len(r.Adjustments) == 0 {
		return errors.New("adjustments is required")
	}

	for i, adj := range r.Adjustments {
		if err := adj.validate(); err != nil {
			return fmt.Errorf("adjustment %d: %w", i, err)
		}
	}

	return nil
}

func (r *createPaymentsAdjustmentsRequest) validate() error {
	if r.Reference == "" {
		return errors.New("reference is required")
	}

	if r.CreatedAt.IsZero() || r.CreatedAt.After(time.Now()) {
		return errors.New("createdAt is empty or in the future")
	}

	if r.Amount == nil {
		return errors.New("amount is required")
	}

	if r.Asset == nil {
		return errors.New("asset is required")
	}

	if r.Status == "" {
		return errors.New("status is required")
	}

	if _, err := models.PaymentStatusFromString(r.Status); err != nil {
		return fmt.Errorf("invalid status: %w", err)
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
			InitialAmount: req.InitialAmount,
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

		for _, adj := range req.Adjustments {
			raw, err := json.Marshal(adj)
			if err != nil {
				otel.RecordError(span, err)
				api.InternalServerError(w, r, err)
				return
			}
			status := models.MustPaymentStatusFromString(adj.Status)

			payment.Adjustments = append(payment.Adjustments, models.PaymentAdjustment{
				ID: models.PaymentAdjustmentID{
					PaymentID: pid,
					Reference: adj.Reference,
					CreatedAt: adj.CreatedAt.UTC(),
					Status:    status,
				},
				PaymentID: pid,
				Reference: adj.Reference,
				CreatedAt: adj.CreatedAt,
				Status:    status,
				Amount:    adj.Amount,
				Asset:     adj.Asset,
				Metadata:  adj.Metadata,
				Raw:       raw,
			})
		}

		err = backend.PaymentsCreate(ctx, payment)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Created(w, payment)
	}
}
