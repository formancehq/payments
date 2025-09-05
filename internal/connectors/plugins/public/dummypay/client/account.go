package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/models"
)

type Account struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	OpeningDate time.Time `json:"opening_date"`
	Currency    string    `json:"currency"`
}

func (c *client) FetchAccounts(ctx context.Context, startToken int, pageSize int) ([]models.PSPAccount, int, error) {
	b, err := c.readFile("accounts.json")
	if err != nil {
		return []models.PSPAccount{}, 0, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	pspAccounts := make([]models.PSPAccount, 0, pageSize)
	if len(b) == 0 {
		return pspAccounts, -1, nil
	}

	accounts := make([]Account, 0)
	err = json.Unmarshal(b, &accounts)
	if err != nil {
		return []models.PSPAccount{}, 0, fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	next := -1
	for i := startToken; i < len(accounts); i++ {
		if len(pspAccounts) >= pageSize {
			if len(accounts)-startToken > len(pspAccounts) {
				next = i
			}
			break
		}

		account := accounts[i]
		pspAccounts = append(pspAccounts, models.PSPAccount{
			Reference:    account.ID,
			CreatedAt:    account.OpeningDate,
			Name:         &account.Name,
			DefaultAsset: &account.Currency,
		})
	}
	return pspAccounts, next, nil
}
