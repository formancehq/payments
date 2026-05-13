package mappers

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
)

func BalanceToPSPBalance(b client.Balance) (models.PSPBalance, error) {
	amount, err := ParseAmount(b.Amount)
	if err != nil {
		return models.PSPBalance{}, fmt.Errorf("balance amount: %w", err)
	}
	return models.PSPBalance{
		AccountReference: b.AccountReference,
		CreatedAt:        b.CreatedAt,
		Amount:           amount,
		Asset:            b.Asset,
	}, nil
}
