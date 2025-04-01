package events

import (
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type V3PaymentInitiationMessagePayload struct {
	// Mandatory fields
	ID          string    `json:"id"`
	ConnectorID string    `json:"connectorID"`
	Provider    string    `json:"provider"`
	Reference   string    `json:"reference"`
	CreatedAt   time.Time `json:"createdAt"`
	ScheduledAt time.Time `json:"scheduledAt"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Amount      *big.Int  `json:"amount"`
	Asset       string    `json:"asset"`

	// Optional fields
	SourceAccountID      string            `json:"sourceAccountID,omitempty"`
	DestinationAccountID string            `json:"destinationAccountID,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

type V3PaymentInitiationAdjustmentMessagePayload struct {
	// Mandatory fields
	ID                  string `json:"id"`
	PaymentInitiationID string `json:"paymentInitiationID"`
	Status              string `json:"status"`

	// Optional fields
	Amount   *big.Int          `json:"amount,omitempty"`
	Asset    *string           `json:"asset,omitempty"`
	Error    *string           `json:"error,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type V3PaymentInitiationRelatedPaymentMessagePayload struct {
	PaymentInitiationID string `json:"paymentInitiationID"`
	PaymentID           string `json:"paymentID"`
}

type V2TransferInitiationsMessagePayload struct {
	ID                   string                                         `json:"id"`
	CreatedAt            time.Time                                      `json:"createdAt"`
	ScheduleAt           time.Time                                      `json:"scheduledAt"`
	ConnectorID          string                                         `json:"connectorId"`
	Provider             string                                         `json:"provider"`
	Description          string                                         `json:"description"`
	Type                 string                                         `json:"type"`
	SourceAccountID      string                                         `json:"sourceAccountId"`
	DestinationAccountID string                                         `json:"destinationAccountId"`
	Amount               *big.Int                                       `json:"amount"`
	Asset                string                                         `json:"asset"`
	Attempts             int                                            `json:"attempts"`
	Status               string                                         `json:"status"`
	Error                string                                         `json:"error"`
	RelatedPayments      []*V2TransferInitiationsPaymentsMessagePayload `json:"relatedPayments"`
}

type V2TransferInitiationsPaymentsMessagePayload struct {
	TransferInitiationID string    `json:"transferInitiationId"`
	PaymentID            string    `json:"paymentId"`
	CreatedAt            time.Time `json:"createdAt"`
	Status               string    `json:"status"`
	Error                string    `json:"error"`
}

func (e Events) NewEventSavedPaymentInitiation(pi models.PaymentInitiation) []publish.EventMessage {
	return []publish.EventMessage{
		toV2TransferInitiationsEvent(pi),
		toV3PaymentInitiationEvent(pi),
	}
}

func toV3PaymentInitiationEvent(pi models.PaymentInitiation) publish.EventMessage {
	payload := V3PaymentInitiationMessagePayload{
		ID:                   pi.ID.String(),
		ConnectorID:          pi.ConnectorID.String(),
		Provider:             pi.ConnectorID.Provider,
		Reference:            pi.Reference,
		CreatedAt:            pi.CreatedAt,
		ScheduledAt:          pi.ScheduledAt,
		Description:          pi.Description,
		Type:                 pi.Type.String(),
		SourceAccountID:      pi.SourceAccountID.String(),
		DestinationAccountID: pi.DestinationAccountID.String(),
		Amount:               pi.Amount,
		Asset:                pi.Asset,
		Metadata:             pi.Metadata,
	}

	return publish.EventMessage{
		IdempotencyKey: pi.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.V3EventTypeSavedPaymentInitiation,
		Payload:        payload,
	}
}

func toV2TransferInitiationsEvent(pi models.PaymentInitiation) publish.EventMessage {
	payload := V2TransferInitiationsMessagePayload{
		ID:                   pi.ID.String(),
		CreatedAt:            pi.CreatedAt,
		ScheduleAt:           pi.ScheduledAt,
		ConnectorID:          pi.ConnectorID.String(),
		Provider:             models.ToV2Provider(pi.ConnectorID.Provider),
		Description:          pi.Description,
		Type:                 pi.Type.String(),
		SourceAccountID:      pi.SourceAccountID.String(),
		DestinationAccountID: pi.DestinationAccountID.String(),
		Amount:               pi.Amount,
		Asset:                pi.Asset,

		// Saved Transfer Initiation has only one attempt
		Attempts: 1,
		Status:   models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION.String(),
	}

	return publish.EventMessage{
		Date:    time.Now().UTC(),
		App:     events.EventApp,
		Version: events.EventVersion,
		Type:    events.V2EventTypeSavedTransferInitiation,
		Payload: payload,
	}
}

func (e Events) NewEventSavedPaymentInitiationAdjustment(adj models.PaymentInitiationAdjustment, pi models.PaymentInitiation) []publish.EventMessage {
	return []publish.EventMessage{
		toV2TransferInitiationAdjustmentEvent(adj, pi),
		toV3PaymentInitiationAdjustmentEvent(adj),
	}
}

func toV3PaymentInitiationAdjustmentEvent(adj models.PaymentInitiationAdjustment) publish.EventMessage {
	payload := V3PaymentInitiationAdjustmentMessagePayload{
		ID:                  adj.ID.String(),
		PaymentInitiationID: adj.ID.PaymentInitiationID.String(),
		Status:              adj.Status.String(),
		Amount:              adj.Amount,
		Asset:               adj.Asset,
		Error: func() *string {
			if adj.Error == nil {
				return nil
			}

			return pointer.For(adj.Error.Error())
		}(),
		Metadata: adj.Metadata,
	}

	return publish.EventMessage{
		IdempotencyKey: adj.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.V3EventTypeSavedPaymentInitiationAdjustment,
		Payload:        payload,
	}
}

func toV2TransferInitiationAdjustmentEvent(adj models.PaymentInitiationAdjustment, pi models.PaymentInitiation) publish.EventMessage {
	payload := V2TransferInitiationsMessagePayload{
		ID:                   pi.ID.String(),
		CreatedAt:            pi.CreatedAt,
		ScheduleAt:           pi.ScheduledAt,
		ConnectorID:          pi.ConnectorID.String(),
		Provider:             models.ToV2Provider(pi.ConnectorID.Provider),
		Description:          pi.Description,
		Type:                 pi.Type.String(),
		SourceAccountID:      pi.SourceAccountID.String(),
		DestinationAccountID: pi.DestinationAccountID.String(),
		Amount:               pi.Amount,
		Asset:                pi.Asset,
		Status:               adj.Status.String(),
		Error: func() string {
			if adj.Error == nil {
				return ""
			}
			return adj.Error.Error()
		}(),
	}

	return publish.EventMessage{
		Date:    time.Now().UTC(),
		App:     events.EventApp,
		Version: events.EventVersion,
		Type:    events.V2EventTypeSavedTransferInitiation,
		Payload: payload,
	}
}

func (e Events) NewEventSavedPaymentInitiationRelatedPayment(
	relatedPayment models.PaymentInitiationRelatedPayments,
	pi models.PaymentInitiation,
	status models.PaymentInitiationAdjustmentStatus) []publish.EventMessage {
	return []publish.EventMessage{
		toV2PaymentInitiationRelatedPayment(relatedPayment, pi, status),
		toV3PaymentInitiationRelatedPayment(relatedPayment),
	}
}

func toV3PaymentInitiationRelatedPayment(relatedPayment models.PaymentInitiationRelatedPayments) publish.EventMessage {
	payload := V3PaymentInitiationRelatedPaymentMessagePayload{
		PaymentInitiationID: relatedPayment.PaymentInitiationID.String(),
		PaymentID:           relatedPayment.PaymentID.String(),
	}

	return publish.EventMessage{
		IdempotencyKey: relatedPayment.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.V3EventTypeSavedPaymentInitiationRelatedPayment,
		Payload:        payload,
	}
}

func toV2PaymentInitiationRelatedPayment(
	relatedPayment models.PaymentInitiationRelatedPayments,
	pi models.PaymentInitiation,
	status models.PaymentInitiationAdjustmentStatus,
) publish.EventMessage {
	payload := V2TransferInitiationsMessagePayload{
		ID:                   pi.ID.String(),
		CreatedAt:            pi.CreatedAt,
		ScheduleAt:           pi.ScheduledAt,
		ConnectorID:          pi.ConnectorID.String(),
		Provider:             models.ToV2Provider(pi.ConnectorID.Provider),
		Description:          pi.Description,
		Type:                 pi.Type.String(),
		SourceAccountID:      pi.SourceAccountID.String(),
		DestinationAccountID: pi.DestinationAccountID.String(),
		Amount:               pi.Amount,
		Asset:                pi.Asset,
		Status:               status.String(),
	}

	payload.RelatedPayments = append(payload.RelatedPayments, &V2TransferInitiationsPaymentsMessagePayload{
		TransferInitiationID: pi.ID.String(),
		PaymentID:            relatedPayment.PaymentID.String(),
		CreatedAt:            pi.CreatedAt,
		Status:               status.String(),
	})

	return publish.EventMessage{
		Date:    time.Now().UTC(),
		App:     events.EventApp,
		Version: events.EventVersion,
		Type:    events.V2EventTypeSavedTransferInitiation,
		Payload: payload,
	}
}
