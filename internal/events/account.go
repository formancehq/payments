package events

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type AccountMessagePayload struct {
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

func (e Events) NewEventSavedAccounts(account models.Account) publish.EventMessage {
	payload := AccountMessagePayload{
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
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedAccounts,
		Payload:        payload,
	}
}
