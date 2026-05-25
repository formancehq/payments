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

// Reference prefixes namespace the three crypto-transactions buckets
// against each other and against user_transactions IDs.
const (
	cryptoDepositPrefix    = "ct-dep:"
	cryptoWithdrawalPrefix = "ct-wd:"
	rippleIOUPrefix        = "ct-iou:"
)

// CryptoKind* labels surface under MetadataKeyType so consumers
// can filter by bucket without re-parsing Raw.
const (
	CryptoKindDeposit    = "deposit"
	CryptoKindWithdrawal = "withdrawal"
	CryptoKindRippleIOU  = "ripple_iou"
)

// CryptoDepositToPSPPayment maps one deposits[] entry to a PSPPayment.
// (nil, nil) on unknown currency — orchestrator logs Info.
func CryptoDepositToPSPPayment(currencies map[string]int, d client.CryptoDeposit) (*models.PSPPayment, error) {
	if d.ID == 0 {
		return nil, fmt.Errorf("crypto deposit: missing id")
	}
	symbol := NormalizeCurrency(d.Currency)
	precision, ok := currencies[symbol]
	if !ok {
		return nil, nil
	}
	amount, err := ParseDecimalAmount(d.Amount.String(), precision)
	if err != nil {
		return nil, fmt.Errorf("crypto deposit %d amount: %w", d.ID, err)
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("crypto deposit %d marshal raw: %w", d.ID, err)
	}
	return &models.PSPPayment{
		Reference: cryptoDepositPrefix + strconv.FormatInt(d.ID, 10),
		CreatedAt: time.Unix(d.Datetime, 0).UTC(),
		Type:      models.PAYMENT_TYPE_PAYIN,
		Amount:    amount,
		Asset:     currency.FormatAsset(currencies, symbol),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    CryptoDepositStatusToPaymentStatus(d.Status),
		Metadata: CryptoTransactionMetadata(
			CryptoKindDeposit, d.Network, d.TxID, d.DestinationAddress, d.PendingReason,
		),
		Raw: raw,
	}, nil
}

// CryptoWithdrawalToPSPPayment — withdrawals have no top-level id +
// no status; reference uses txid, status is always SUCCEEDED (the
// endpoint only surfaces processed rows).
func CryptoWithdrawalToPSPPayment(currencies map[string]int, w client.CryptoWithdrawal) (*models.PSPPayment, error) {
	if w.TxID == "" {
		return nil, fmt.Errorf("crypto withdrawal: missing txid")
	}
	symbol := NormalizeCurrency(w.Currency)
	precision, ok := currencies[symbol]
	if !ok {
		return nil, nil
	}
	amount, err := ParseDecimalAmount(w.Amount.String(), precision)
	if err != nil {
		return nil, fmt.Errorf("crypto withdrawal %s amount: %w", w.TxID, err)
	}
	raw, err := json.Marshal(w)
	if err != nil {
		return nil, fmt.Errorf("crypto withdrawal %s marshal raw: %w", w.TxID, err)
	}
	return &models.PSPPayment{
		Reference: cryptoWithdrawalPrefix + w.TxID,
		CreatedAt: time.Unix(w.Datetime, 0).UTC(),
		Type:      models.PAYMENT_TYPE_PAYOUT,
		Amount:    amount,
		Asset:     currency.FormatAsset(currencies, symbol),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Metadata: CryptoTransactionMetadata(
			CryptoKindWithdrawal, w.Network, w.TxID, w.DestinationAddress, "",
		),
		Raw: raw,
	}, nil
}

// RippleIOUToPSPPayment — same shape as CryptoWithdrawal; emitted
// as PAYOUT so IOU debits match wire semantics.
func RippleIOUToPSPPayment(currencies map[string]int, r client.RippleIOUTransaction) (*models.PSPPayment, error) {
	if r.TxID == "" {
		return nil, fmt.Errorf("ripple IOU: missing txid")
	}
	symbol := NormalizeCurrency(r.Currency)
	precision, ok := currencies[symbol]
	if !ok {
		return nil, nil
	}
	amount, err := ParseDecimalAmount(r.Amount.String(), precision)
	if err != nil {
		return nil, fmt.Errorf("ripple IOU %s amount: %w", r.TxID, err)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("ripple IOU %s marshal raw: %w", r.TxID, err)
	}
	return &models.PSPPayment{
		Reference: rippleIOUPrefix + r.TxID,
		CreatedAt: time.Unix(r.Datetime, 0).UTC(),
		Type:      models.PAYMENT_TYPE_PAYOUT,
		Amount:    amount,
		Asset:     currency.FormatAsset(currencies, symbol),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Metadata: CryptoTransactionMetadata(
			CryptoKindRippleIOU, r.Network, r.TxID, r.DestinationAddress, "",
		),
		Raw: raw,
	}, nil
}
