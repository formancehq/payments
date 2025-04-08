package events

import (
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type BalanceMessagePayload struct {
	AccountID     string    `json:"accountID"`
	ConnectorID   string    `json:"connectorID"`
	Provider      string    `json:"provider"`
	CreatedAt     time.Time `json:"createdAt"`
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
	Asset         string    `json:"asset"`
	Balance       *big.Int  `json:"balance"`
}

func (e Events) NewEventSavedBalances(balance models.Balance) publish.EventMessage {
	payload := BalanceMessagePayload{
		AccountID:     balance.AccountID.String(),
		ConnectorID:   balance.AccountID.ConnectorID.String(),
		Provider:      models.ToV3Provider(balance.AccountID.ConnectorID.Provider),
		CreatedAt:     balance.CreatedAt,
		LastUpdatedAt: balance.LastUpdatedAt,
		Asset:         balance.Asset,
		Balance:       balance.Balance,
	}

	return publish.EventMessage{
		IdempotencyKey: balance.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedBalances,
		Payload:        payload,
	}
}
