package events

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type V3PaymentMessagePayload struct {
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

type V2PaymentMessagePayload struct {
	ID                   string               `json:"id"`
	Reference            string               `json:"reference"`
	CreatedAt            time.Time            `json:"createdAt"`
	ConnectorID          string               `json:"connectorId"`
	Provider             string               `json:"provider"`
	Type                 models.PaymentType   `json:"type"`
	Status               models.PaymentStatus `json:"status"`
	Scheme               models.PaymentScheme `json:"scheme"`
	Asset                string               `json:"asset"`
	SourceAccountID      string               `json:"sourceAccountId,omitempty"`
	DestinationAccountID string               `json:"destinationAccountId,omitempty"`
	Links                []api.Link           `json:"links"`
	RawData              json.RawMessage      `json:"rawData"`

	InitialAmount *big.Int          `json:"initialAmount"`
	Amount        *big.Int          `json:"amount"`
	Metadata      map[string]string `json:"metadata"`
}

func (e Events) NewEventSavedPayments(payment models.Payment, adjustment models.PaymentAdjustment) []publish.EventMessage {
	return []publish.EventMessage{
		e.toV2PaymentEvent(payment, adjustment),
		e.toV3PaymentEvent(payment, adjustment),
	}
}

func (e Events) toV3PaymentEvent(payment models.Payment, adjustment models.PaymentAdjustment) publish.EventMessage {
	payload := V3PaymentMessagePayload{
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
		Version:        events.V3EventVersion,
		Type:           events.V3EventTypeSavedPayments,
		Payload:        payload,
	}
}

func (e Events) toV2PaymentEvent(payment models.Payment, adjustment models.PaymentAdjustment) publish.EventMessage {
	payload := V2PaymentMessagePayload{
		ID:            payment.ID.String(),
		Reference:     payment.Reference,
		Type:          payment.Type,
		Status:        payment.Status,
		InitialAmount: payment.InitialAmount,
		Amount:        payment.Amount,
		Scheme:        payment.Scheme,
		Asset:         payment.Asset,
		CreatedAt:     payment.CreatedAt,
		ConnectorID:   payment.ConnectorID.String(),
		Provider:      models.ToV2Provider(payment.ConnectorID.Provider),
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
			URI:  e.stackURL + "/api/payments/accounts/" + payment.SourceAccountID.String(),
		})
	}

	if payment.DestinationAccountID != nil {
		payload.Links = append(payload.Links, api.Link{
			Name: "destination_account",
			URI:  e.stackURL + "/api/payments/accounts/" + payment.DestinationAccountID.String(),
		})
	}

	return publish.EventMessage{
		Date:    time.Now().UTC(),
		App:     events.EventApp,
		Version: events.V2EventVersion,
		Type:    events.V2EventTypeSavedPayments,
		Payload: payload,
	}
}
