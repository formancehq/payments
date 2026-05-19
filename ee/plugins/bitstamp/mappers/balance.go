package mappers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

// AccountBalanceToPSPBalance derives a PSPBalance from the AccountBalance
// payload already snapshotted on the parent PSPAccount.Raw. This is the
// Qonto-style FromPayload pattern (MAPPINGS.md §3.2 / §2.1) — no extra
// /api/v2/account_balances/ call per cycle.
//
// observedAt should be the orchestrator's "now" so adjacent test runs
// produce deterministic CreatedAt values.
//
// Returns nil for unknown currencies (caller skips), matching the
// connector's invariant that accounts and balances agree on the
// supported-currency set.
func AccountBalanceToPSPBalance(currencies map[string]int, account models.PSPAccount, observedAt time.Time) (*models.PSPBalance, error) {
	if account.Raw == nil {
		return nil, fmt.Errorf("missing account raw payload for %s", account.Reference)
	}
	var bal client.AccountBalance
	if err := json.Unmarshal(account.Raw, &bal); err != nil {
		return nil, fmt.Errorf("unmarshal account raw for %s: %w", account.Reference, err)
	}
	symbol := NormalizeCurrency(bal.Currency)
	precision, known := currencies[symbol]
	if !known {
		return nil, nil
	}
	amount, err := ParseAmount(bal.Available, precision)
	if err != nil {
		return nil, fmt.Errorf("balance for %s: %w", symbol, err)
	}
	return &models.PSPBalance{
		AccountReference: account.Reference,
		Asset:            FormatAsset(currencies, symbol),
		Amount:           amount,
		CreatedAt:        observedAt,
	}, nil
}
