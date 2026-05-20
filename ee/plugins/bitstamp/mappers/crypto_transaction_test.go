package mappers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

// localCurrencies covers the assets used in the test fixtures —
// declared per-file to avoid relying on test-suite-wide state.
var cryptoTestCurrencies = map[string]int{
	"BTC":  8,
	"USDC": 6,
	"ETH":  18,
}

func TestCryptoDepositToPSPPayment_Pending(t *testing.T) {
	t.Parallel()

	d := client.CryptoDeposit{
		ID:                 42,
		Network:            "bitcoin",
		Currency:           "BTC",
		TxID:               "abc123",
		Amount:             json.Number("1.23"),
		Datetime:           1759995000, // Oct 9 2025
		Status:             "PENDING",
		PendingReason:      "ADDRESS_VERIFICATION_NEEDED",
		DestinationAddress: "1A1zP1eP",
	}
	got, err := CryptoDepositToPSPPayment(cryptoTestCurrencies, d)
	if err != nil {
		t.Fatalf("CryptoDepositToPSPPayment: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil PSPPayment for known currency")
	}
	if !strings.HasPrefix(got.Reference, "ct-dep:") {
		t.Errorf("Reference must use ct-dep: prefix, got %q", got.Reference)
	}
	if got.Type != models.PAYMENT_TYPE_PAYIN {
		t.Errorf("Type = %v, want PAYIN", got.Type)
	}
	if got.Status != models.PAYMENT_STATUS_PENDING {
		t.Errorf("Status = %v, want PENDING", got.Status)
	}
	if got.Asset != "BTC/8" {
		t.Errorf("Asset = %q, want BTC/8", got.Asset)
	}
	if got.Amount.Int64() != 123_000_000 { // 1.23 BTC at precision 8
		t.Errorf("Amount = %s, want 123_000_000", got.Amount)
	}
	if got.Metadata[MetadataKeySource] != PaymentSourceCryptoTransactions {
		t.Errorf("Source metadata missing/wrong: %q", got.Metadata[MetadataKeySource])
	}
	if got.Metadata[MetadataKeyType] != CryptoKindDeposit {
		t.Errorf("Type metadata = %q, want %q", got.Metadata[MetadataKeyType], CryptoKindDeposit)
	}
	if got.Metadata[MetadataKeyNetwork] != "bitcoin" {
		t.Errorf("Network metadata missing")
	}
	if got.Metadata[MetadataKeyTxID] != "abc123" {
		t.Errorf("TxID metadata missing")
	}
	if got.Metadata[MetadataKeyDestinationAddress] != "1A1zP1eP" {
		t.Errorf("DestinationAddress metadata missing")
	}
	if got.Metadata[MetadataKeyPendingReason] != "ADDRESS_VERIFICATION_NEEDED" {
		t.Errorf("PendingReason metadata missing")
	}
	if len(got.Raw) == 0 {
		t.Error("Raw must be populated")
	}
}

func TestCryptoDepositToPSPPayment_CompletedOmitsPendingReason(t *testing.T) {
	t.Parallel()

	d := client.CryptoDeposit{
		ID:       43,
		Network:  "bitcoin",
		Currency: "BTC",
		TxID:     "def456",
		Amount:   json.Number("0.5"),
		Datetime: 1759995100,
		Status:   "COMPLETED",
		// PendingReason intentionally empty
	}
	got, err := CryptoDepositToPSPPayment(cryptoTestCurrencies, d)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Status != models.PAYMENT_STATUS_SUCCEEDED {
		t.Errorf("Status = %v, want SUCCEEDED", got.Status)
	}
	if _, present := got.Metadata[MetadataKeyPendingReason]; present {
		t.Errorf("pending_reason must NOT be emitted on COMPLETED deposits, got %+v", got.Metadata)
	}
}

func TestCryptoDepositToPSPPayment_UnknownStatusMapsToUnknown(t *testing.T) {
	t.Parallel()

	d := client.CryptoDeposit{
		ID: 44, Network: "bitcoin", Currency: "BTC", TxID: "x", Amount: json.Number("0.1"),
		Datetime: 1, Status: "WHATEVER_THE_FUTURE_BRINGS",
	}
	got, err := CryptoDepositToPSPPayment(cryptoTestCurrencies, d)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Status != models.PAYMENT_STATUS_UNKNOWN {
		t.Errorf("unknown status must map to UNKNOWN (never silent SUCCEEDED), got %v", got.Status)
	}
}

func TestCryptoDepositToPSPPayment_UnknownCurrencyReturnsNil(t *testing.T) {
	t.Parallel()

	d := client.CryptoDeposit{
		ID: 45, Currency: "FUTURE_COIN", TxID: "y", Amount: json.Number("1"), Datetime: 1, Status: "COMPLETED",
	}
	got, err := CryptoDepositToPSPPayment(cryptoTestCurrencies, d)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("unknown currency must return nil (orchestrator logs Warn), got %+v", got)
	}
}

func TestCryptoDepositToPSPPayment_BadAmountReturnsError(t *testing.T) {
	t.Parallel()

	d := client.CryptoDeposit{
		ID: 46, Currency: "BTC", TxID: "z", Amount: json.Number("not-a-decimal"), Datetime: 1, Status: "PENDING",
	}
	_, err := CryptoDepositToPSPPayment(cryptoTestCurrencies, d)
	if err == nil {
		t.Error("expected error on bad amount string")
	}
}

func TestCryptoDepositToPSPPayment_MissingIDReturnsError(t *testing.T) {
	t.Parallel()
	_, err := CryptoDepositToPSPPayment(cryptoTestCurrencies, client.CryptoDeposit{Currency: "BTC", TxID: "x"})
	if err == nil {
		t.Error("expected error on missing id")
	}
}

func TestCryptoWithdrawalToPSPPayment(t *testing.T) {
	t.Parallel()

	w := client.CryptoWithdrawal{
		Currency:           "BTC",
		Network:            "bitcoin",
		DestinationAddress: "3FiK",
		TxID:               "wd-tx-1",
		Amount:             json.Number("0.00012"),
		Datetime:           1642665114,
	}
	got, err := CryptoWithdrawalToPSPPayment(cryptoTestCurrencies, w)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.HasPrefix(got.Reference, "ct-wd:") {
		t.Errorf("Reference must use ct-wd: prefix, got %q", got.Reference)
	}
	if got.Type != models.PAYMENT_TYPE_PAYOUT {
		t.Errorf("Type = %v, want PAYOUT", got.Type)
	}
	if got.Status != models.PAYMENT_STATUS_SUCCEEDED {
		t.Errorf("withdrawals have no status field — must be SUCCEEDED, got %v", got.Status)
	}
	if got.Metadata[MetadataKeyType] != CryptoKindWithdrawal {
		t.Errorf("kind metadata = %q, want %q", got.Metadata[MetadataKeyType], CryptoKindWithdrawal)
	}
}

func TestCryptoWithdrawalToPSPPayment_MissingTxIDReturnsError(t *testing.T) {
	t.Parallel()
	_, err := CryptoWithdrawalToPSPPayment(cryptoTestCurrencies, client.CryptoWithdrawal{Currency: "BTC"})
	if err == nil {
		t.Error("expected error on missing txid (the natural primary key)")
	}
}

func TestRippleIOUToPSPPayment(t *testing.T) {
	t.Parallel()
	r := client.RippleIOUTransaction{
		Currency: "BTC", Network: "bitcoin", DestinationAddress: "3FiK",
		TxID: "iou-tx-1", Amount: json.Number("0.001"), Datetime: 1642665114,
	}
	got, err := RippleIOUToPSPPayment(cryptoTestCurrencies, r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.HasPrefix(got.Reference, "ct-iou:") {
		t.Errorf("Reference must use ct-iou: prefix, got %q", got.Reference)
	}
	if got.Type != models.PAYMENT_TYPE_PAYOUT {
		t.Errorf("Type = %v, want PAYOUT", got.Type)
	}
	if got.Metadata[MetadataKeyType] != CryptoKindRippleIOU {
		t.Errorf("kind metadata = %q, want %q", got.Metadata[MetadataKeyType], CryptoKindRippleIOU)
	}
}
