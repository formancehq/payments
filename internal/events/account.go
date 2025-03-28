package events

import (
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type AccountMessagePayload struct {
	ID           string            `json:"id"`
	CreatedAt    time.Time         `json:"createdAt"`
	Reference    string            `json:"reference"`
	Provider     string            `json:"provider"`
	ConnectorID  string            `json:"connectorId"`
	DefaultAsset string            `json:"defaultAsset"`
	AccountName  string            `json:"accountName"`
	Type         string            `json:"type"`
	Metadata     map[string]string `json:"metadata"`
	RawData      json.RawMessage   `json:"rawData"`
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
		payload.AccountName = *account.Name
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
