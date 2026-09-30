package v2

import (
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/transport/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/common"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

type transferInitiationResponse struct {
	ID                   string            `json:"id"`
	Reference            string            `json:"reference"`
	CreatedAt            time.Time         `json:"createdAt"`
	ScheduledAt          time.Time         `json:"scheduledAt"`
	Description          string            `json:"description"`
	SourceAccountID      string            `json:"sourceAccountID"`
	DestinationAccountID string            `json:"destinationAccountID"`
	ConnectorID          string            `json:"connectorID"`
	Provider             string            `json:"provider"`
	Type                 string            `json:"type"`
	Amount               *big.Int          `json:"amount"`
	InitialAmount        *big.Int          `json:"initialAmount"`
	Asset                string            `json:"asset"`
	Status               string            `json:"status"`
	Error                string            `json:"error"`
	Metadata             map[string]string `json:"metadata"`
}

type transferInitiationPaymentsResponse struct {
	PaymentID string    `json:"paymentID"`
	CreatedAt time.Time `json:"createdAt"`
	Status    string    `json:"status"`
}

type transferInitiationAdjustmentsResponse struct {
	AdjustmentID string            `json:"adjustmentID"`
	CreatedAt    time.Time         `json:"createdAt"`
	Status       string            `json:"status"`
	Error        string            `json:"error"`
	Metadata     map[string]string `json:"metadata"`
}

type readTransferInitiationResponse struct {
	transferInitiationResponse
	RelatedPayments    []transferInitiationPaymentsResponse    `json:"relatedPayments"`
	RelatedAdjustments []transferInitiationAdjustmentsResponse `json:"relatedAdjustments"`
}

func transferInitiationsGet(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_transferInitiationsGet")
		defer span.End()

		span.SetAttributes(attribute.String("transferInitiationID", transferInitiationID(r)))
		id, err := models.PaymentInitiationIDFromString(transferInitiationID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		transferInitiation, err := backend.PaymentInitiationsGet(ctx, id)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		relatedPayments, err := backend.PaymentInitiationRelatedPaymentsListAll(ctx, id)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		relatedAdjustments, err := backend.PaymentInitiationAdjustmentsListAll(ctx, id)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		t := translatePaymentInitiationToResponse(transferInitiation)
		if len(relatedAdjustments) > 0 {
			t.Status = translateLastStatus(relatedAdjustments[0].Status)
			t.Error = func() string {
				if relatedAdjustments[0].Error == nil {
					return ""
				}
				return relatedAdjustments[0].Error.Error()
			}()
		}

		resp := &readTransferInitiationResponse{
			transferInitiationResponse: t,
			RelatedPayments:            translateRelatedPayments(relatedPayments),
			RelatedAdjustments:         translateAdjustments(relatedAdjustments),
		}

		err = json.NewEncoder(w).Encode(api.BaseResponse[readTransferInitiationResponse]{
			Data: resp,
		})
		if err != nil {
			otel.RecordError(span, err)
			common.InternalServerError(w, r, err)
			return
		}
	}
}

func translateAdjustments(from []models.PaymentInitiationAdjustment) []transferInitiationAdjustmentsResponse {
	to := make([]transferInitiationAdjustmentsResponse, len(from))
	for i, adjustment := range from {
		status, toSend := translateStatus(adjustment.Status)
		if !toSend {
			continue
		}
		to[i] = transferInitiationAdjustmentsResponse{
			AdjustmentID: adjustment.ID.String(),
			CreatedAt:    adjustment.CreatedAt,
			Status:       status,
			Error: func() string {
				if adjustment.Error == nil {
					return ""
				}
				return adjustment.Error.Error()
			}(),
			Metadata: adjustment.Metadata,
		}
	}
	return to
}

// translateLastStatus renders a payment initiation's current status for v2
// clients, whose TransferInitiationStatus enum is narrower than v3's: statuses
// introduced in v3 are reported as their closest v2 equivalent.
func translateLastStatus(from models.PaymentInitiationAdjustmentStatus) string {
	switch from {
	case models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_SCHEDULED_FOR_PROCESSING:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING.String()
	case models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_NOT_INITIATED:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED.String()
	default:
		return from.String()
	}
}

func translateStatus(from models.PaymentInitiationAdjustmentStatus) (string, bool) {
	switch from {
	case models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_SCHEDULED_FOR_PROCESSING:
		// PAYMENT_INITIATION_ADJUSTMENT_STATUS_SCHEDULED_FOR_PROCESSING is not supported
		// in v2 as it is introduced in v3. Since we're gonna list all adjustments
		// we can drop this one
		return "", false
	case models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_NOT_INITIATED:
		// PAYMENT_INITIATION_ADJUSTMENT_STATUS_NOT_INITIATED is not supported in v2
		// either, but unlike the one above it carries a failure: dropping it would
		// hide the rejection from v2 clients, which used to see it as FAILED.
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED.String(), true
	default:
		return from.String(), true
	}
}

func translateRelatedPayments(from []models.Payment) []transferInitiationPaymentsResponse {
	to := make([]transferInitiationPaymentsResponse, len(from))
	for i, payment := range from {
		to[i] = transferInitiationPaymentsResponse{
			PaymentID: payment.ID.String(),
			CreatedAt: payment.CreatedAt,
			Status:    payment.Status.String(),
		}
	}
	return to
}

func translatePaymentInitiationToResponse(from *models.PaymentInitiation) transferInitiationResponse {
	return transferInitiationResponse{
		ID:                   from.ID.String(),
		Reference:            from.Reference,
		CreatedAt:            from.CreatedAt,
		ScheduledAt:          from.ScheduledAt,
		Description:          from.Description,
		SourceAccountID:      from.SourceAccountID.String(),
		DestinationAccountID: from.DestinationAccountID.String(),
		ConnectorID:          from.ConnectorID.String(),
		Provider:             toV2Provider(from.ConnectorID.Provider),
		Type:                 from.Type.String(),
		Amount:               from.Amount,
		InitialAmount:        from.Amount,
		Asset:                from.Asset,
		Metadata:             from.Metadata,
	}
}
