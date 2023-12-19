package messages

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type paymentMessagePayload struct {
	ID                   string               `json:"id"`
	Reference            string               `json:"reference"`
	CreatedAt            time.Time            `json:"createdAt"`
	ConnectorID          string               `json:"connectorId"`
	Provider             string               `json:"provider"`
	Type                 models.PaymentType   `json:"type"`
	Status               models.PaymentStatus `json:"status"`
	Scheme               models.PaymentScheme `json:"scheme"`
	Asset                models.Asset         `json:"asset"`
	SourceAccountID      *models.AccountID    `json:"sourceAccountId,omitempty"`
	DestinationAccountID *models.AccountID    `json:"destinationAccountId,omitempty"`
	RawData              json.RawMessage      `json:"rawData"`

	// TODO: Remove 'initialAmount' once frontend has switched to 'amount
	InitialAmount *big.Int          `json:"initialAmount"`
	Amount        *big.Int          `json:"amount"`
	Metadata      map[string]string `json:"metadata"`
}

func NewEventSavedPayments(provider models.ConnectorProvider, payment *models.Payment) events.EventMessage {
	payload := paymentMessagePayload{
		ID:                   payment.ID.String(),
		Reference:            payment.Reference,
		Type:                 payment.Type,
		Status:               payment.Status,
		InitialAmount:        payment.InitialAmount,
		Amount:               payment.Amount,
		Scheme:               payment.Scheme,
		Asset:                payment.Asset,
		CreatedAt:            payment.CreatedAt,
		ConnectorID:          payment.ConnectorID.String(),
		Provider:             provider.String(),
		SourceAccountID:      payment.SourceAccountID,
		DestinationAccountID: payment.DestinationAccountID,
		RawData:              payment.RawData,
		Metadata: func() map[string]string {
			ret := make(map[string]string)
			for _, m := range payment.Metadata {
				ret[m.Key] = m.Value
			}
			return ret
		}(),
	}

	return events.EventMessage{
		Date:    time.Now().UTC(),
		App:     events.EventApp,
		Version: events.EventVersion,
		Type:    events.EventTypeSavedPayments,
		Payload: payload,
	}
}
