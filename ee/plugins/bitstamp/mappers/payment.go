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
//
// Transfer rows (types 14 / 33 / 35) are remapped to sign-based
// PAYOUT (negative amount → this connector is the source) /
// PAYIN (positive amount → this connector is the destination) per
// MAPPINGS §3.6. The pair-id metadata (= tx.id stringified) is what
// lets a downstream consumer join the two legs across connectors.
func UserTransactionToPSPPayment(currencies map[string]int, tx client.UserTransaction) (PaymentMapResult, error) {
	if tx.HasDerivativesMarker() {
		return PaymentMapResult{Skip: true, DerivativesRow: true}, nil
	}
	if IsOrderOrConversion(tx.Type) {
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

	paymentType, metadata := classifyPaymentType(tx)

	return PaymentMapResult{
		Payment: &models.PSPPayment{
			Reference: strconv.FormatInt(tx.ID, 10),
			CreatedAt: createdAt,
			Type:      paymentType,
			Amount:    amount,
			Asset:     asset,
			Scheme:    models.PAYMENT_SCHEME_OTHER,
			// user_transactions returns settled-only history; pending
			// withdrawals live on /withdrawal-requests/ which a sibling
			// orchestrator polls (MAPPINGS.md §3.3.c).
			Status:   models.PAYMENT_STATUS_SUCCEEDED,
			Metadata: metadata,
			Raw:      raw,
		},
		UnknownType: !IsKnownTransactionType(tx.Type),
	}, nil
}

// classifyPaymentType maps a user_transactions row to a
// (PaymentType, metadata). Two-legged transfer types (14 / 33 / 35)
// split by amount sign — see MAPPINGS §4.3 sub-section on
// cross-account transfers. Counterparty fields are absent on the
// wire today; the metadata builder omits the keys when empty.
func classifyPaymentType(tx client.UserTransaction) (models.PaymentType, map[string]string) {
	base := PaymentMetadata(tx)
	if !IsTransferType(tx.Type) {
		paymentType, _ := TransactionTypeToPaymentType(tx.Type)
		return paymentType, base
	}
	direction := TransferDirectionIncoming
	paymentType := models.PAYMENT_TYPE_PAYIN
	if transferAmountIsNegative(tx) {
		direction = TransferDirectionOutgoing
		paymentType = models.PAYMENT_TYPE_PAYOUT
	}
	pair := TransferPairMetadata(tx.ID, direction, "", "")
	return paymentType, MergeMetadata(base, pair)
}

func transferAmountIsNegative(tx client.UserTransaction) bool {
	for _, value := range tx.CurrencyAmounts {
		if value == "" || IsZeroAmount(AbsAmount(value)) {
			continue
		}
		return IsNegative(value)
	}
	return false
}

