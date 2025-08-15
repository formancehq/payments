package events

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type paymentMessagePayload struct {
	// Mandatory fields
	ID            string          `json:"id"`
	ConnectorID   string          `json:"connectorID"`
	Provider      string          `json:"provider"`
	Reference     string          `json:"reference"`
	CreatedAt     time.Time       `json:"createdAt"`
	Type          string          `json:"type"`
	Status        string          `json:"status"`
	Scheme        string          `json:"scheme"`
	Asset         string          `json:"asset"`
	RawData       json.RawMessage `json:"rawData"`
	InitialAmount *big.Int        `json:"initialAmount"`
	Amount        *big.Int        `json:"amount"`

	// Optional fields
	SourceAccountID      string            `json:"sourceAccountID,omitempty"`
	DestinationAccountID string            `json:"destinationAccountID,omitempty"`
	Links                []api.Link        `json:"links,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
}

func (e Events) NewEventSavedPayments(payment models.Payment, adjustment models.PaymentAdjustment) publish.EventMessage {
	payload := paymentMessagePayload{
		ID:            payment.ID.String(),
		Reference:     payment.Reference,
		Type:          payment.Type.String(),
		Status:        payment.Status.String(),
		InitialAmount: payment.InitialAmount,
		Amount:        payment.Amount,
		Scheme:        payment.Scheme.String(),
		Asset:         payment.Asset,
		CreatedAt:     payment.CreatedAt,
		ConnectorID:   payment.ConnectorID.String(),
		Provider:      models.ToV3Provider(payment.ConnectorID.Provider),
		SourceAccountID: func() string {
			if payment.SourceAccountID == nil {
				return ""
			}
			return payment.SourceAccountID.String()
		}(),
		DestinationAccountID: func() string {
			if payment.DestinationAccountID == nil {
				return ""
			}
			return payment.DestinationAccountID.String()
		}(),
		RawData:  adjustment.Raw,
		Metadata: payment.Metadata,
	}

	if payment.SourceAccountID != nil {
		payload.Links = append(payload.Links, api.Link{
			Name: "source_account",
			URI:  e.stackURL + "/api/payments/v3/accounts/" + payment.SourceAccountID.String(),
		})
	}

	if payment.DestinationAccountID != nil {
		payload.Links = append(payload.Links, api.Link{
			Name: "destination_account",
			URI:  e.stackURL + "/api/payments/v3/accounts/" + payment.DestinationAccountID.String(),
		})
	}

	return publish.EventMessage{
		IdempotencyKey: adjustment.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedPayments,
		Payload:        payload,
	}
}

type paymentDeletedMessagePayload struct {
	ID string `json:"id"`
}

func (e Events) NewEventPaymentDeleted(paymentID models.PaymentID) publish.EventMessage {
	payload := paymentDeletedMessagePayload{
		ID: paymentID.String(),
	}

	return publish.EventMessage{
		IdempotencyKey: fmt.Sprintf("delete:%s", paymentID.String()),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeDeletedPayments,
		Payload:        payload,
	}
}
