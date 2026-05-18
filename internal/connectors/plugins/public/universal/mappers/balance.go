package mappers

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
)

func BalanceToPSPBalance(b client.Balance) (models.PSPBalance, error) {
	if err := requireRef("balance.accountReference", b.AccountReference); err != nil {
		return models.PSPBalance{}, err
	}
	amount, err := ParseAmount(b.Amount)
	if err != nil {
		return models.PSPBalance{}, fmt.Errorf("balance %s amount: %w", b.AccountReference, err)
	}
	if amount == nil {
		return models.PSPBalance{}, fmt.Errorf("balance %s: missing amount", b.AccountReference)
	}
	if b.Asset == "" {
		return models.PSPBalance{}, fmt.Errorf("balance %s: missing asset", b.AccountReference)
	}
	return models.PSPBalance{
		AccountReference: b.AccountReference,
		CreatedAt:        b.CreatedAt,
		Amount:           amount,
		Asset:            b.Asset,
	}, nil
}
