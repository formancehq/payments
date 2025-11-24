package events

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type PaymentInitiationMessagePayload struct {
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

func (p *PaymentInitiationMessagePayload) MarshalJSON() ([]byte, error) {
	type Alias PaymentInitiationMessagePayload
	var amountStr *string
	if p.Amount != nil {
		s := p.Amount.String()
		amountStr = &s
	}
	return json.Marshal(&struct {
		Amount *string `json:"amount"`
		*Alias
	}{
		Amount: amountStr,
		Alias:  (*Alias)(p),
	})
}

func (p *PaymentInitiationMessagePayload) UnmarshalJSON(data []byte) error {
	type Alias PaymentInitiationMessagePayload
	aux := &struct {
		Amount *string `json:"amount"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Amount != nil {
		bi := new(big.Int)
		if _, ok := bi.SetString(*aux.Amount, 10); !ok {
			return fmt.Errorf("invalid amount string: %s", *aux.Amount)
		}
		p.Amount = bi
	} else {
		p.Amount = nil
	}
	return nil
}

type PaymentInitiationAdjustmentMessagePayload struct {
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

func (p *PaymentInitiationAdjustmentMessagePayload) MarshalJSON() ([]byte, error) {
	type Alias PaymentInitiationAdjustmentMessagePayload
	var amountStr *string
	if p.Amount != nil {
		s := p.Amount.String()
		amountStr = &s
	}
	return json.Marshal(&struct {
		Amount *string `json:"amount,omitempty"`
		*Alias
	}{
		Amount: amountStr,
		Alias:  (*Alias)(p),
	})
}

func (p *PaymentInitiationAdjustmentMessagePayload) UnmarshalJSON(data []byte) error {
	type Alias PaymentInitiationAdjustmentMessagePayload
	aux := &struct {
		Amount *string `json:"amount,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Amount != nil {
		bi := new(big.Int)
		if _, ok := bi.SetString(*aux.Amount, 10); !ok {
			return fmt.Errorf("invalid amount string: %s", *aux.Amount)
		}
		p.Amount = bi
	} else {
		p.Amount = nil
	}
	return nil
}

type PaymentInitiationRelatedPaymentMessagePayload struct {
	PaymentInitiationID string `json:"paymentInitiationID"`
	PaymentID           string `json:"paymentID"`
}

func (e Events) NewEventSavedPaymentInitiation(pi models.PaymentInitiation) publish.EventMessage {
	payload := PaymentInitiationMessagePayload{
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
		Type:           events.EventTypeSavedPaymentInitiation,
		Payload:        payload,
	}
}

func (e Events) NewEventSavedPaymentInitiationAdjustment(adj models.PaymentInitiationAdjustment) publish.EventMessage {
	payload := PaymentInitiationAdjustmentMessagePayload{
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
		Type:           events.EventTypeSavedPaymentInitiationAdjustment,
		Payload:        payload,
	}
}

func (e Events) NewEventSavedPaymentInitiationRelatedPayment(relatedPayment models.PaymentInitiationRelatedPayments) publish.EventMessage {
	payload := PaymentInitiationRelatedPaymentMessagePayload{
		PaymentInitiationID: relatedPayment.PaymentInitiationID.String(),
		PaymentID:           relatedPayment.PaymentID.String(),
	}

	return publish.EventMessage{
		IdempotencyKey: relatedPayment.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedPaymentInitiationRelatedPayment,
		Payload:        payload,
	}
}
