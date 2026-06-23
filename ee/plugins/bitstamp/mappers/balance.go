package mappers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

// AccountBalanceToPSPBalance derives a PSPBalance from the AccountBalance
// snapshotted on the parent PSPAccount.Raw (Qonto FromPayload pattern —
// no extra API call). observedAt should be the orchestrator's "now".
// (nil, nil) on unknown currency.
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
	amount, err := ParseDecimalAmount(bal.Available, precision)
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
