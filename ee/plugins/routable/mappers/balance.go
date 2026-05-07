package mappers

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/models"
)

// AccountToBalance converts a Routable settings account's available
// amount into a PSPBalance. Pending balances are intentionally not
// surfaced as a second entry: the Formance balance model represents one
// snapshot per (account, asset) pair, and "available" is the canonical
// signal.
func AccountToBalance(a client.Account, now time.Time) (models.PSPBalance, error) {
	currencyCode := a.CurrencyCode
	if currencyCode == "" {
		// Routable historically defaulted USD here; keep the same
		// fallback behaviour rather than dropping balances silently.
		currencyCode = "USD"
	}
	precision, err := PrecisionFor(currencyCode)
	if err != nil {
		return models.PSPBalance{}, err
	}
	amount, err := ToMinorUnits(a.TypeDetails.AvailableAmount, precision)
	if err != nil {
		return models.PSPBalance{}, fmt.Errorf("parsing available_amount: %w", err)
	}
	return models.PSPBalance{
		AccountReference: a.ID,
		Asset:            FormatAsset(currencyCode),
		Amount:           amount,
		CreatedAt:        now,
	}, nil
}
