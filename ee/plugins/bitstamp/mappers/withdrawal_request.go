package mappers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

const withdrawalRequestPrefix = "wr:"

// withdrawal-requests timestamps have no microseconds (unlike
// user_transactions which uses BitstampDatetimeLayout).
const withdrawalRequestDatetimeLayout = "2006-01-02 15:04:05"

// WithdrawalRequestToPSPPayment maps one withdrawal-requests/ row to
// a PSPPayment. Always PAYOUT. (nil, nil) on unknown currency.
func WithdrawalRequestToPSPPayment(currencies map[string]int, w client.WithdrawalRequest) (*models.PSPPayment, error) {
	if w.ID == 0 {
		return nil, fmt.Errorf("withdrawal request: missing id")
	}
	symbol := NormalizeCurrency(w.Currency)
	precision, ok := currencies[symbol]
	if !ok {
		return nil, nil
	}
	amount, err := parseDecimalAmount(w.Amount, precision)
	if err != nil {
		return nil, fmt.Errorf("withdrawal request %d amount: %w", w.ID, err)
	}
	createdAt, err := time.Parse(withdrawalRequestDatetimeLayout, w.Datetime)
	if err != nil {
		return nil, fmt.Errorf("withdrawal request %d datetime %q: %w", w.ID, w.Datetime, err)
	}
	raw, err := json.Marshal(w)
	if err != nil {
		return nil, fmt.Errorf("withdrawal request %d marshal raw: %w", w.ID, err)
	}
	return &models.PSPPayment{
		Reference: withdrawalRequestPrefix + strconv.FormatInt(w.ID, 10),
		CreatedAt: createdAt.UTC(),
		Type:      models.PAYMENT_TYPE_PAYOUT,
		Amount:    amount,
		Asset:     currency.FormatAsset(currencies, symbol),
		Scheme:    WithdrawalRequestTypeToScheme(w.Type),
		Status:    WithdrawalRequestStatusToPaymentStatus(w.Status),
		Metadata: WithdrawalRequestMetadata(
			strconv.Itoa(w.Type), w.Network, w.Address, w.TxID, w.TransactionID,
		),
		Raw: raw,
	}, nil
}
