package models

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"

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

	// Optional, can be filled if the balance is related to an open banking connector
	PsuID *uuid.UUID
	// Optional, can be filled if the balance is related to an open banking connector
	OpenBankingConnectionID *string
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

	PsuID                   *uuid.UUID `json:"psuID"`
	OpenBankingConnectionID *string    `json:"openBankingConnectionID,omitempty"`
}

func (b *Balance) IdempotencyKey() string {
	var ik = struct {
		AccountID     string
		CreatedAt     int64
		LastUpdatedAt int64
		Asset         string
	}{
		AccountID:     b.AccountID.String(),
		CreatedAt:     b.CreatedAt.UnixNano(),
		LastUpdatedAt: b.LastUpdatedAt.UnixNano(),
		Asset:         b.Asset,
	}
	return IdempotencyKey(ik)
}

func (b Balance) MarshalJSON() ([]byte, error) {
	var psuId string
	if b.PsuID != nil {
		psuId = b.PsuID.String()
	}

	var openBankingConnectionId *string
	if b.OpenBankingConnectionID != nil {
		openBankingConnectionId = b.OpenBankingConnectionID
	}

	return json.Marshal(&struct {
		AccountID     string    `json:"accountID"`
		CreatedAt     time.Time `json:"createdAt"`
		LastUpdatedAt time.Time `json:"lastUpdatedAt"`

		Asset   string   `json:"asset"`
		Balance *big.Int `json:"balance"`

		PsuId                   string  `json:"psuID"`
		OpenBankingConnectionID *string `json:"openBankingConnectionID,omitempty"`
	}{
		AccountID:               b.AccountID.String(),
		CreatedAt:               b.CreatedAt,
		LastUpdatedAt:           b.LastUpdatedAt,
		Asset:                   b.Asset,
		Balance:                 b.Balance,
		PsuId:                   psuId,
		OpenBankingConnectionID: openBankingConnectionId,
	})
}

func (b *Balance) UnmarshalJSON(data []byte) error {
	var aux struct {
		AccountID               string    `json:"accountID"`
		CreatedAt               time.Time `json:"createdAt"`
		LastUpdatedAt           time.Time `json:"lastUpdatedAt"`
		Asset                   string    `json:"asset"`
		Balance                 *big.Int  `json:"balance"`
		PSUID                   string    `json:"psuID"`
		OpenBankingConnectionID *string   `json:"openBankingConnectionID,omitempty"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	accountID, err := AccountIDFromString(aux.AccountID)
	if err != nil {
		return err
	}
	if aux.PSUID != "" {
		PSUID, err := uuid.Parse(aux.PSUID)
		if err != nil {
			return err
		}
		b.PsuID = &PSUID
	}

	b.AccountID = accountID
	b.CreatedAt = aux.CreatedAt
	b.LastUpdatedAt = aux.LastUpdatedAt
	b.Asset = aux.Asset
	b.Balance = aux.Balance
	b.OpenBankingConnectionID = aux.OpenBankingConnectionID

	return nil
}

type AggregatedBalance struct {
	Asset           string      `json:"asset"`
	Amount          *big.Int    `json:"amount"`
	RelatedAccounts []AccountID `json:"relatedAccounts"`
}

func (a AggregatedBalance) MarshalJSON() ([]byte, error) {
	relatedAccounts := make([]string, len(a.RelatedAccounts))
	for i := range a.RelatedAccounts {
		relatedAccounts[i] = a.RelatedAccounts[i].String()
	}

	return json.Marshal(&struct {
		Asset           string   `json:"asset"`
		Amount          *big.Int `json:"amount"`
		RelatedAccounts []string `json:"relatedAccounts"`
	}{
		Asset:           a.Asset,
		Amount:          a.Amount,
		RelatedAccounts: relatedAccounts,
	})
}

func (a *AggregatedBalance) UnmarshalJSON(data []byte) error {
	var aux struct {
		Asset           string   `json:"asset"`
		Amount          *big.Int `json:"amount"`
		RelatedAccounts []string `json:"relatedAccounts"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	a.Asset = aux.Asset
	a.Amount = aux.Amount
	a.RelatedAccounts = make([]AccountID, len(aux.RelatedAccounts))
	for i := range aux.RelatedAccounts {
		accountID, err := AccountIDFromString(aux.RelatedAccounts[i])
		if err != nil {
			return err
		}
		a.RelatedAccounts[i] = accountID
	}

	return nil
}

func FromPSPBalance(from PSPBalance, connectorID ConnectorID, psuId *uuid.UUID, openBankingConnectionId *string) (Balance, error) {
	if err := from.Validate(); err != nil {
		return Balance{}, err
	}

	if psuId == nil {
		psuId = from.PsuID
	}
	if openBankingConnectionId == nil {
		openBankingConnectionId = from.OpenBankingConnectionID
	}
	return Balance{
		AccountID: AccountID{
			Reference:   from.AccountReference,
			ConnectorID: connectorID,
		},
		CreatedAt:               from.CreatedAt,
		LastUpdatedAt:           from.CreatedAt,
		Asset:                   from.Asset,
		Balance:                 from.Amount,
		PsuID:                   psuId,
		OpenBankingConnectionID: openBankingConnectionId,
	}, nil
}

func FromPSPBalances(from []PSPBalance, connectorID ConnectorID, psuId *uuid.UUID, openBankingConnectionId *string) ([]Balance, error) {
	balances := make([]Balance, 0, len(from))
	for _, b := range from {
		balance, err := FromPSPBalance(b, connectorID, psuId, openBankingConnectionId)
		if err != nil {
			return nil, err
		}
		balances = append(balances, balance)
	}
	return balances, nil
}
