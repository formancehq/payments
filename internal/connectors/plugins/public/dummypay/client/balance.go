package client

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
)

type Balance struct {
	AccountID      string `json:"account_id"`
	AmountInMinors int64  `json:"amount_in_minors"`
	Currency       string `json:"currency"`
}

func (c *client) FetchBalance(ctx context.Context, accountID string) (*models.PSPBalance, error) {
	b, err := c.readFile("balances.json")
	if err != nil {
		return &models.PSPBalance{}, fmt.Errorf("failed to fetch balances: %w", err)
	}

	balances := make([]Balance, 0)
	if len(b) == 0 {
		return nil, nil
	}

	err = json.Unmarshal(b, &balances)
	if err != nil {
		return &models.PSPBalance{}, fmt.Errorf("failed to unmarshal balances: %w", err)
	}

	for _, balance := range balances {
		if balance.AccountID != accountID {
			continue
		}
		return &models.PSPBalance{
			AccountReference: balance.AccountID,
			CreatedAt:        time.Now().Truncate(time.Second),
			Asset:            balance.Currency,
			Amount:           big.NewInt(balance.AmountInMinors),
		}, nil
	}
	return &models.PSPBalance{}, nil
}
