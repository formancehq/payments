package mappers

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/models"
)

// PaymentMapResult tells the orchestrator how to handle the row.
//   - Payment != nil → emit it.
//   - Skip == true   → the row belongs to another pipeline (orders /
//     conversions) or is intentionally ignored.
//   - UnknownType    → emit as OTHER and Infof the ledger id (the
//     logging interface has no Warnf level).
type PaymentMapResult struct {
	Payment     *models.PSPPayment
	Skip        bool
	UnknownType bool
}

// LedgerEntryToPSPPayment maps a single ledger row into a PSPPayment.
// Trade / order / conversion rows are skipped here — they belong to
// the orders + conversions pipelines. wallets maps a normalised symbol
// → spot account reference; the payment is attributed to that account
// (PAYIN → destination, PAYOUT → source) so it links to a real
// PSPAccount, with the raw variant kept in kraken_asset metadata.
func LedgerEntryToPSPPayment(currencies map[string]int, wallets map[string]string, ledgerID string, e client.LedgerEntry) (PaymentMapResult, error) {
	kind, paymentType, signDriven := ClassifyLedgerType(e.Type)
	if kind != LedgerKindPayment {
		return PaymentMapResult{Skip: true}, nil
	}

	symbol := NormalizeAsset(e.Asset)
	if symbol == "" {
		return PaymentMapResult{Skip: true}, nil
	}
	precision, known := currencies[symbol]
	if !known {
		// Unknown asset: skip silently. The asset cache TTL refresh
		// will pick it up on subsequent cycles.
		return PaymentMapResult{Skip: true}, nil
	}

	if IsZeroAmount(e.Amount) {
		return PaymentMapResult{Skip: true}, nil
	}

	if signDriven && IsNegative(e.Amount) {
		paymentType = models.PAYMENT_TYPE_PAYOUT
	}

	amount, err := ParseDecimalAmount(AbsAmount(e.Amount), precision)
	if err != nil {
		return PaymentMapResult{}, fmt.Errorf("ledger %s amount: %w", ledgerID, err)
	}

	raw, err := json.Marshal(e)
	if err != nil {
		return PaymentMapResult{}, fmt.Errorf("ledger %s marshal: %w", ledgerID, err)
	}

	payment := &models.PSPPayment{
		Reference: ledgerID,
		CreatedAt: FloatEpochToTime(e.Time),
		Type:      paymentType,
		Amount:    amount,
		Asset:     FormatAsset(currencies, symbol),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Metadata:  LedgerMetadata(ledgerID, e),
		Raw:       raw,
	}
	// PAYIN credits the destination, PAYOUT debits the source; OTHER is
	// left unattributed (ambiguous direction). Refs are optional, so an
	// unresolved spot account stays nil.
	if ref := spotRef(wallets, symbol); ref != nil {
		switch paymentType {
		case models.PAYMENT_TYPE_PAYIN:
			payment.DestinationAccountReference = ref
		case models.PAYMENT_TYPE_PAYOUT:
			payment.SourceAccountReference = ref
		}
	}

	return PaymentMapResult{
		Payment:     payment,
		UnknownType: !IsKnownLedgerType(e.Type),
	}, nil
}
