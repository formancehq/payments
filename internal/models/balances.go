package models

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/utils/assets"
)

type PSPBalance struct {
	// PSP account reference of the balance.
	AccountReference string

	// Balance Creation date.
	CreatedAt time.Time

	// Balance amount.
	Amount *big.Int

	// Currency. Should be in minor currencies unit.
	// For example: USD/2
	Asset string
}

func (p *PSPBalance) Validate() error {
	if p.AccountReference == "" {
		return fmt.Errorf("missing account reference: %w", ErrValidation)
	}

	if p.CreatedAt.IsZero() {
		return fmt.Errorf("missing balance createdAt: %w", ErrValidation)
	}

	if p.Amount == nil {
		return fmt.Errorf("missing balance amount: %w", ErrValidation)
	}

	if !assets.IsValid(p.Asset) {
		return fmt.Errorf("invalid balance asset: %w", ErrValidation)
	}

	return nil
}

type Balance struct {
	// Balance related formance account id
	AccountID AccountID `json:"accountID"`
	// Balance created at
	CreatedAt time.Time `json:"createdAt"`
	// Balance last updated at
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`

	// Currency. Should be in minor currencies unit.
	Asset string `json:"asset"`
	// Balance amount.
	Balance *big.Int `json:"balance"`
}

func (b *Balance) IdempotencyKey() string {
	var ik = struct {
		AccountID     string
		CreatedAt     int64
		LastUpdatedAt int64
	}{
		AccountID:     b.AccountID.String(),
		CreatedAt:     b.CreatedAt.UnixNano(),
		LastUpdatedAt: b.LastUpdatedAt.UnixNano(),
	}
	return IdempotencyKey(ik)
}

func (b Balance) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		AccountID     string    `json:"accountID"`
		CreatedAt     time.Time `json:"createdAt"`
		LastUpdatedAt time.Time `json:"lastUpdatedAt"`

		Asset   string   `json:"asset"`
		Balance *big.Int `json:"balance"`
	}{
		AccountID:     b.AccountID.String(),
		CreatedAt:     b.CreatedAt,
		LastUpdatedAt: b.LastUpdatedAt,
		Asset:         b.Asset,
		Balance:       b.Balance,
	})
}

func (b *Balance) UnmarshalJSON(data []byte) error {
	var aux struct {
		AccountID     string    `json:"accountID"`
		CreatedAt     time.Time `json:"createdAt"`
		LastUpdatedAt time.Time `json:"lastUpdatedAt"`
		Asset         string    `json:"asset"`
		Balance       *big.Int  `json:"balance"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	accountID, err := AccountIDFromString(aux.AccountID)
	if err != nil {
		return err
	}

	b.AccountID = accountID
	b.CreatedAt = aux.CreatedAt
	b.LastUpdatedAt = aux.LastUpdatedAt
	b.Asset = aux.Asset
	b.Balance = aux.Balance

	return nil
}

type AggregatedBalance struct {
	Asset  string   `json:"asset"`
	Amount *big.Int `json:"amount"`
}

func FromPSPBalance(from PSPBalance, connectorID ConnectorID) (Balance, error) {
	if err := from.Validate(); err != nil {
		return Balance{}, err
	}

	return Balance{
		AccountID: AccountID{
			Reference:   from.AccountReference,
			ConnectorID: connectorID,
		},
		CreatedAt:     from.CreatedAt,
		LastUpdatedAt: from.CreatedAt,
		Asset:         from.Asset,
		Balance:       from.Amount,
	}, nil
}

func FromPSPBalances(from []PSPBalance, connectorID ConnectorID) ([]Balance, error) {
	balances := make([]Balance, 0, len(from))
	for _, b := range from {
		balance, err := FromPSPBalance(b, connectorID)
		if err != nil {
			return nil, err
		}
		balances = append(balances, balance)
	}
	return balances, nil
}
