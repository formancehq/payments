package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Account struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
}

func (c *client) GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_accounts")

	if page > 1 {
		return []*Account{}, nil
	}
	if c.sdk == nil || c.entityID == "" {
		return nil, fmt.Errorf("checkout sdk not initialized or missing entityID")
	}

	entity, err := c.sdk.Accounts.GetEntity(c.entityID)
	if err != nil {
		return nil, fmt.Errorf("checkout.accounts.getEntity(%s): %w", c.entityID, err)
	}

	if b, err := json.MarshalIndent(entity, "", "  "); err == nil {
		fmt.Printf("Received entity from Checkout: %s\n", string(b))
	}

	id := c.entityID
	name := fmt.Sprint(entity.Company.LegalName)
	status := fmt.Sprint(entity.Status)

	accounts := []*Account{{
		ID:       id,
		Name:     name,
		Status:   status,
	}}

	if b, _ := json.Marshal(accounts); true {
		fmt.Printf("[checkout] GetAccounts returns %d account(s): %s\n", len(accounts), string(b))
	}

	return accounts, nil
}
