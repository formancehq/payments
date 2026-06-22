package mappers

import (
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

func AccountToBalance(a client.Account, now time.Time) (models.PSPBalance, error) {
	currencyCode := a.CurrencyCode
	if currencyCode == "" {
		// Routable historically defaulted USD here; preserve it rather
		// than dropping balances on accounts with an empty currency code.
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
