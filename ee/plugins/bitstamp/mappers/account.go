package mappers

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

// AccountBalanceToPSPAccount maps one row from /api/v2/account_balances/
// to a PSPAccount. Returns (nil, nil) when the row's currency cannot
// be normalised; rows with all-zero Available/Total/Reserved are
// filtered upstream by the orchestrator (MAPPINGS.md §3.1).
//
// The raw AccountBalance JSON is preserved in PSPAccount.Raw so the
// FromPayload-driven balances task (§3.2) can derive the balance
// snapshot without a second API call.
func AccountBalanceToPSPAccount(currencies map[string]int, bal client.AccountBalance) (*models.PSPAccount, error) {
	symbol := NormalizeCurrency(bal.Currency)
	if symbol == "" {
		return nil, nil
	}
	raw, err := json.Marshal(bal)
	if err != nil {
		return nil, fmt.Errorf("marshal account balance for %s: %w", symbol, err)
	}
	account := models.PSPAccount{
		Reference: symbol,
		CreatedAt: BitstampGenesis,
		Raw:       raw,
	}
	if _, known := currencies[symbol]; known {
		asset := FormatAsset(currencies, symbol)
		account.DefaultAsset = &asset
	}
	return &account, nil
}
