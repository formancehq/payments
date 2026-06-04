package mappers

import (
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/models"
)

// RawBalanceToPSPBalance derives a PSPBalance row from one raw Kraken
// BalanceEx variant. AccountReference is the raw code (one account per
// asset class), Asset is the normalised symbol, Amount is the available
// balance per Kraken docs: balance + credit - credit_used - hold_trade
// (clamped to >= 0). credit / credit_used are populated on VIP/Pro
// accounts with a credit line. No aggregation: distinct account
// references mean no (account, asset) collision. Unknown assets / fully
// empty rows yield (nil, nil) so the caller can skip silently.
func RawBalanceToPSPBalance(currencies map[string]int, rawCode string, entry client.BalanceExEntry, observedAt time.Time) (*models.PSPBalance, error) {
	symbol := NormalizeAsset(rawCode)
	if symbol == "" {
		return nil, nil
	}
	precision, known := currencies[symbol]
	if !known {
		return nil, nil
	}
	if IsZeroAmount(entry.Balance) && IsZeroAmount(entry.HoldTrade) && IsZeroAmount(entry.Credit) {
		return nil, nil
	}
	available, err := AvailableBalance(entry.Balance, entry.HoldTrade, entry.Credit, entry.CreditUsed, precision)
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
