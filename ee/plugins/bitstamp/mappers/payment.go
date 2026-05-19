package mappers

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

// PaymentMapResult tells the orchestrator how to handle the row.
//   - Payment != nil — emit it.
//   - Skip = true   — silently skip (orders / conversions / multi-asset
//                     and similar known non-payment shapes).
//   - DerivativesRow = true — log Warn before skipping; spot-only stance.
//   - UnknownType = true    — log Warn (with tx.id) before emitting as
//                             PAYMENT_TYPE_OTHER so a new code surfaces.
type PaymentMapResult struct {
	Payment        *models.PSPPayment
	Skip           bool
	DerivativesRow bool
	UnknownType    bool
}

// UserTransactionToPSPPayment maps a user_transactions row to a
// PSPPayment. Returns Skip=true for trade/conversion rows (handled by
// orders and conversions respectively) and for any row that does not
// reduce to exactly one non-zero known currency.
//
// Spot-only stance: rows carrying derivatives markers in their Raw
// (margin_mode / leverage_rate) are flagged DerivativesRow=true and
// skipped — the orchestrator logs Warn with the row ID so a
// derivatives-enabled account surfaces the gap loudly.
func UserTransactionToPSPPayment(currencies map[string]int, tx client.UserTransaction) (PaymentMapResult, error) {
	if tx.HasDerivativesMarker() {
		return PaymentMapResult{Skip: true, DerivativesRow: true}, nil
	}
	if IsOrderOrConversion(tx.Type) {
		return PaymentMapResult{Skip: true}, nil
	}

	paymentType, ok := TransactionTypeToPaymentType(tx.Type)
	if !ok {
		// Defensive: TransactionTypeToPaymentType returns ok=false only
		// for order/conversion rows, which we filter above. Reaching
		// here would indicate a regression.
		return PaymentMapResult{Skip: true}, nil
	}

	asset, amount, hasAmount, err := ResolveSinglePaymentAsset(currencies, tx.CurrencyAmounts)
	if err != nil {
		return PaymentMapResult{}, fmt.Errorf("resolve amount for tx %d: %w", tx.ID, err)
	}
	if !hasAmount {
		return PaymentMapResult{Skip: true}, nil
	}

	createdAt, err := ParseBitstampTime(tx.Datetime)
	if err != nil {
		return PaymentMapResult{}, fmt.Errorf("payment tx %d: %w", tx.ID, err)
	}

	raw, err := json.Marshal(tx)
	if err != nil {
		return PaymentMapResult{}, fmt.Errorf("marshal raw for tx %d: %w", tx.ID, err)
	}

	return PaymentMapResult{
		Payment: &models.PSPPayment{
			Reference: strconv.FormatInt(tx.ID, 10),
			CreatedAt: createdAt,
			Type:      paymentType,
			Amount:    amount,
			Asset:     asset,
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			// user_transactions returns settled-only history; pending
			// withdrawals live on /withdrawal-requests/ which the
			// connector does not poll today (MAPPINGS.md §7).
			Status:   models.PAYMENT_STATUS_SUCCEEDED,
			Metadata: PaymentMetadata(tx),
			Raw:      raw,
		},
		UnknownType: !IsKnownTransactionType(tx.Type),
	}, nil
}

