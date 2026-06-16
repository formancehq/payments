package mappers

import (
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/models"
)

// IncludeBalanceEntry is the single inclusion rule shared by
// fetchNextAccounts and fetchNextBalances: a BalanceEx variant is emitted
// iff its raw code normalizes to a symbol present in the /Assets cache.
// Zero balances are intentionally NOT filtered — Kraken only returns a
// row for an asset the account holds (or has held), and emitting both an
// account and a balance from this same predicate guarantees a balance can
// never reference an account that was not emitted. Returns the normalized
// symbol on success.
func IncludeBalanceEntry(currencies map[string]int, rawCode string) (symbol string, ok bool) {
	symbol = NormalizeAsset(rawCode)
	if symbol == "" {
		return "", false
	}
	if _, known := currencies[symbol]; !known {
		return "", false
	}
	return symbol, true
}

// RawBalanceToPSPBalance derives a PSPBalance row from one raw Kraken
// BalanceEx variant. AccountReference is the raw code (one account per
// asset class), Asset is the normalised symbol, Amount is the available
// balance per Kraken docs: balance + credit - credit_used - hold_trade
// (clamped to >= 0). credit / credit_used are populated on VIP/Pro
// accounts with a credit line. No aggregation: distinct account
// references mean no (account, asset) collision. Unsupported assets yield
// (nil, nil) so the caller can skip silently.
func RawBalanceToPSPBalance(currencies map[string]int, rawCode string, entry client.BalanceExEntry, observedAt time.Time) (*models.PSPBalance, error) {
	symbol, ok := IncludeBalanceEntry(currencies, rawCode)
	if !ok {
		return nil, nil
	}
	available, err := AvailableBalance(entry.Balance, entry.HoldTrade, entry.Credit, entry.CreditUsed, currencies[symbol])
	if err != nil {
		return nil, fmt.Errorf("balance for %s: %w", rawCode, err)
	}
	return &models.PSPBalance{
		AccountReference: strings.ToUpper(strings.TrimSpace(rawCode)),
		Asset:            FormatAsset(currencies, symbol),
		Amount:           available,
		CreatedAt:        observedAt,
	}, nil
}
