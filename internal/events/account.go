package events

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type V3AccountMessagePayload struct {
	// Mandatory fields
	ID          string          `json:"id"`
	Provider    string          `json:"provider"`
	ConnectorID string          `json:"connectorID"`
	CreatedAt   time.Time       `json:"createdAt"`
	Reference   string          `json:"reference"`
	Type        string          `json:"type"`
	RawData     json.RawMessage `json:"rawData"`

	// Optional fields
	DefaultAsset string            `json:"defaultAsset,omitempty"`
	Name         string            `json:"name,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type V2AccountMessagePayload struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"createdAt"`
	Reference    string    `json:"reference"`
	ConnectorID  string    `json:"connectorId"`
	Provider     string    `json:"provider"`
	DefaultAsset string    `json:"defaultAsset"`
	AccountName  string    `json:"accountName"`
	Type         string    `json:"type"`
}

func (e Events) NewEventSavedAccounts(account models.Account) []publish.EventMessage {
	return []publish.EventMessage{
		toV2AccountEvent(account),
		toV3AccountEvent(account),
	}
}

func toV3AccountEvent(account models.Account) publish.EventMessage {
	payload := V3AccountMessagePayload{
		ID:          account.ID.String(),
		ConnectorID: account.ConnectorID.String(),
		Provider:    models.ToV3Provider(account.ConnectorID.Provider),
		CreatedAt:   account.CreatedAt,
		Reference:   account.Reference,
		Type:        string(account.Type),
		Metadata:    account.Metadata,
		RawData:     account.Raw,
	}

	if account.DefaultAsset != nil {
		payload.DefaultAsset = *account.DefaultAsset
	}

	if account.Name != nil {
		payload.Name = *account.Name
	}

	return publish.EventMessage{
		IdempotencyKey: account.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.V3EventVersion,
		Type:           events.V3EventTypeSavedAccounts,
		Payload:        payload,
	}
}

func toV2AccountEvent(account models.Account) publish.EventMessage {
	payload := V2AccountMessagePayload{
		ID:          account.ID.String(),
		CreatedAt:   account.CreatedAt,
		Reference:   account.Reference,
		ConnectorID: account.ConnectorID.String(),
		Provider:    models.ToV2Provider(account.ConnectorID.Provider),
		Type:        string(account.Type),
	}

	if account.Name != nil {
		payload.AccountName = *account.Name
	}

	if account.DefaultAsset != nil {
		payload.DefaultAsset = *account.DefaultAsset
	}

	return publish.EventMessage{
		IdempotencyKey: account.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.V2EventVersion,
		Type:           events.V2EventTypeSavedAccounts,
		Payload:        payload,
	}
}
