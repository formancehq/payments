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
//   - UnknownAsset   → the asset is missing from the cache (likely listed
//     after the last refresh); the orchestrator should refresh + retry
//     before the watermark advances rather than drop the row.
//   - UnknownType    → emit as OTHER and Infof the ledger id (the
//     logging interface has no Warnf level).
type PaymentMapResult struct {
	Payment      *models.PSPPayment
	Skip         bool
	UnknownAsset bool
	UnknownType  bool
}

// LedgerEntryToPSPPayment maps a single ledger row into a PSPPayment.
// Trade / order / conversion rows are skipped here — they belong to
// the orders + conversions pipelines. wallets maps a normalised symbol
// → spot account reference; the payment is attributed to that account
// (PAYIN → destination, PAYOUT → source) so it links to a real
// PSPAccount, with the raw variant kept in kraken_asset metadata.
func LedgerEntryToPSPPayment(currencies map[string]int, wallets map[string]string, ledgerID string, e client.LedgerEntry) (PaymentMapResult, error) {
	kind, paymentType := ClassifyLedgerType(e.Type)
	if kind != LedgerKindPayment {
		return PaymentMapResult{Skip: true}, nil
	}

	symbol := NormalizeAsset(e.Asset)
	if symbol == "" {
		return PaymentMapResult{Skip: true}, nil
	}
	precision, known := currencies[symbol]
	if !known {
		// Asset not in the cache — likely listed after the last refresh.
		// Flag it so the orchestrator refreshes + retries before advancing
		// the watermark, instead of permanently skipping the row.
		return PaymentMapResult{Skip: true, UnknownAsset: true}, nil
	}

	if IsZeroAmount(e.Amount) {
		return PaymentMapResult{Skip: true}, nil
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
		Metadata:  LedgerMetadata(e),
		Raw:       raw,
	}
	// Attribute the spot account by amount sign: a negative amount leaves
	// the account (source), a positive one enters it (destination). This
	// holds for PAYOUT/PAYIN and for the TRANSFER's known (spot) leg; the
	// counterparty wallet (futures/staking/subaccount) isn't tracked, so
	// the other side stays nil. Refs are optional.
	if ref := spotRef(wallets, symbol); ref != nil {
		if IsNegative(e.Amount) {
			payment.SourceAccountReference = ref
		} else {
			payment.DestinationAccountReference = ref
		}
	}

	return PaymentMapResult{
		Payment:     payment,
		UnknownType: !IsKnownLedgerType(e.Type),
	}, nil
}
