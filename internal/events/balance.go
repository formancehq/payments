package events

import (
	"encoding/json"
	"fmt"
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

func (b *BalanceMessagePayload) MarshalJSON() ([]byte, error) {
	type Alias BalanceMessagePayload
	var balanceStr *string
	if b.Balance != nil {
		s := b.Balance.String()
		balanceStr = &s
	}
	return json.Marshal(&struct {
		Balance *string `json:"balance"`
		*Alias
	}{
		Balance: balanceStr,
		Alias:   (*Alias)(b),
	})
}

func (b *BalanceMessagePayload) UnmarshalJSON(data []byte) error {
	type Alias BalanceMessagePayload
	aux := &struct {
		Balance *string `json:"balance"`
		*Alias
	}{
		Alias: (*Alias)(b),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Balance != nil {
		bi := new(big.Int)
		if _, ok := bi.SetString(*aux.Balance, 10); !ok {
			return fmt.Errorf("invalid balance string: %s", *aux.Balance)
		}
		b.Balance = bi
	} else {
		b.Balance = nil
	}
	return nil
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
