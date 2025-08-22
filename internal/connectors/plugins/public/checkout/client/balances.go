package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/checkout/checkout-sdk-go/balances"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Balance struct {
	Descriptor        string   `json:"descriptor"`
	CurrencyAccountID string   `json:"currencyAccountId"`
	Currency          string   `json:"currency"`
	Available  		  int64    `json:"available"`
	Pending           int64    `json:"pending"`
	Payable           int64    `json:"payable"`
	Collateral        int64    `json:"collateral"`
}

func (c *client) GetAccountBalances(ctx context.Context) ([]*Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_account_balances")

	if c.sdk == nil || c.entityID == "" {
		return nil, fmt.Errorf("checkout sdk not initialized or missing entityID")
	}

	resp, err := c.sdk.Balances.RetrieveEntityBalances(
		c.entityID,
		balances.QueryFilter{WithCurrencyAccountId: true},
	)
	if err != nil {
		return nil, fmt.Errorf("checkout.accounts.getEntityBalances(%s): %w", c.entityID, err)
	}

	balances := make([]*Balance, 0, len(resp.Data))
	for _, ab := range resp.Data {
		balances = append(balances, &Balance{
			Descriptor: 	   ab.Descriptor,
			CurrencyAccountID: ab.CurrencyAccountId,
			Currency:          ab.HoldingCurrency,
			Available:         ab.Balances.Available,
			Pending:           ab.Balances.Pending,
			Payable:           ab.Balances.Payable,
			Collateral:        ab.Balances.Collateral,
		})
	}

	if b, _ := json.Marshal(balances); true {
		fmt.Printf("[checkout] GetAccountBalances returns %d balance(s): %s\n", len(balances), string(b))
	}

	return balances, nil
}
