package mappers

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

func newTx(raw string) client.UserTransaction {
	var tx client.UserTransaction
	if err := json.Unmarshal([]byte(raw), &tx); err != nil {
		panic(err)
	}
	return tx
}

func TestUserTransactionToPSPPaymentDeposit(t *testing.T) {
	t.Parallel()
	tx := newTx(`{
		"id": 1001,
		"datetime": "2025-09-25 14:42:59.000000",
		"type": "0",
		"fee": "0.00",
		"btc": "0.5",
		"usd": "0.00"
	}`)
	res, err := UserTransactionToPSPPayment(testCurrencies, tx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Skip || res.Payment == nil {
		t.Fatalf("expected payment, got %#v", res)
	}
	if res.Payment.Reference != "1001" {
		t.Errorf("reference=%q, want 1001", res.Payment.Reference)
	}
	if res.Payment.Type != models.PAYMENT_TYPE_PAYIN {
		t.Errorf("type=%v, want PAYIN", res.Payment.Type)
	}
	if res.Payment.Asset != "BTC/8" {
		t.Errorf("asset=%q, want BTC/8", res.Payment.Asset)
	}
	if res.Payment.Amount.Cmp(big.NewInt(50000000)) != 0 {
		t.Errorf("amount=%s, want 50000000", res.Payment.Amount)
	}
	if res.Payment.Status != models.PAYMENT_STATUS_SUCCEEDED {
		t.Errorf("status=%v, want SUCCEEDED", res.Payment.Status)
	}
	if res.Payment.Metadata[MetadataKeyType] != "0" {
		t.Errorf("missing type metadata: %v", res.Payment.Metadata)
	}
	if res.UnknownType {
		t.Errorf("type 0 should be known")
	}
}

func TestUserTransactionToPSPPaymentWithdrawalNegative(t *testing.T) {
	t.Parallel()
	tx := newTx(`{
		"id": 1002,
		"datetime": "2025-09-25 14:42:59.000000",
		"type": "1",
		"fee": "0.01",
		"eur": "-25.50"
	}`)
	res, err := UserTransactionToPSPPayment(testCurrencies, tx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Payment == nil {
		t.Fatal("expected payment")
	}
	if res.Payment.Type != models.PAYMENT_TYPE_PAYOUT {
		t.Errorf("type=%v, want PAYOUT", res.Payment.Type)
	}
	// Amount must be positive (CLAUDE.md convention) even though the
	// raw value is signed negative.
	if res.Payment.Amount.Cmp(big.NewInt(2550)) != 0 {
		t.Errorf("amount=%s, want 2550 (positive)", res.Payment.Amount)
	}
	if res.Payment.Metadata[MetadataKeyFee] != "0.01" {
		t.Errorf("missing fee metadata: %v", res.Payment.Metadata)
	}
}

func TestUserTransactionToPSPPaymentSkipsTradesAndConversions(t *testing.T) {
	t.Parallel()
	for _, txType := range []string{TxTypeMarketTrade, TxTypeBuySell} {
		tx := newTx(`{
			"id": 2000,
			"datetime": "2025-09-25 14:42:59.000000",
			"type": "` + txType + `",
			"fee": "0",
			"eur": "-5.00",
			"usdc": "5.81"
		}`)
		res, err := UserTransactionToPSPPayment(testCurrencies, tx)
		if err != nil {
			t.Fatalf("err for type %s: %v", txType, err)
		}
		if !res.Skip || res.Payment != nil {
			t.Errorf("type %s should be skipped, got %#v", txType, res)
		}
	}
}

func TestUserTransactionToPSPPaymentSkipsDerivatives(t *testing.T) {
	t.Parallel()
	tx := newTx(`{
		"id": 3000,
		"datetime": "2025-09-25 14:42:59.000000",
		"type": "0",
		"fee": "0",
		"btc": "1.0",
		"margin_mode": "FLEXIBLE",
		"leverage_rate": "5"
	}`)
	res, err := UserTransactionToPSPPayment(testCurrencies, tx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.Skip || !res.DerivativesRow {
		t.Errorf("expected DerivativesRow skip, got %#v", res)
	}
}

func TestUserTransactionToPSPPaymentUnknownTypeIsEmitted(t *testing.T) {
	t.Parallel()
	tx := newTx(`{
		"id": 4000,
		"datetime": "2025-09-25 14:42:59.000000",
		"type": "999",
		"fee": "0",
		"btc": "0.1"
	}`)
	res, err := UserTransactionToPSPPayment(testCurrencies, tx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Payment == nil {
		t.Fatal("unknown type should still emit a PAYMENT_TYPE_OTHER row so it isn't lost")
	}
	if res.Payment.Type != models.PAYMENT_TYPE_OTHER {
		t.Errorf("type=%v, want OTHER", res.Payment.Type)
	}
	if !res.UnknownType {
		t.Error("UnknownType=true is the orchestrator's Warn signal")
	}
}

func TestUserTransactionToPSPPaymentSkipsTwoAssetRow(t *testing.T) {
	t.Parallel()
	// A non-type-36 row that still presents two non-zero known
	// currencies (defensive) — the payment mapper refuses to guess
	// which leg is "the amount".
	tx := newTx(`{
		"id": 5000,
		"datetime": "2025-09-25 14:42:59.000000",
		"type": "33",
		"fee": "0",
		"eur": "-5.00",
		"usdc": "5.81"
	}`)
	res, err := UserTransactionToPSPPayment(testCurrencies, tx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.Skip || res.Payment != nil {
		t.Errorf("two-asset payment row must skip, got %#v", res)
	}
}

// TestUserTransactionToPSPPayment_TransferOutgoing asserts the sign-
// based remap for type-14 / 33 / 35 rows where the local account is
// the source (negative amount).
func TestUserTransactionToPSPPayment_TransferOutgoing(t *testing.T) {
	t.Parallel()
	cases := []string{"14", "33", "35"}
	for _, txType := range cases {
		txType := txType
		t.Run("type_"+txType, func(t *testing.T) {
			t.Parallel()
			tx := newTx(`{
				"id": 12345,
				"datetime": "2025-09-25 14:42:59.000000",
				"type": "` + txType + `",
				"fee": "0",
				"btc": "-0.50000000"
			}`)
			res, err := UserTransactionToPSPPayment(testCurrencies, tx)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if res.Skip || res.Payment == nil {
				t.Fatalf("transfer row must emit a payment, got %#v", res)
			}
			if res.Payment.Type != models.PAYMENT_TYPE_PAYOUT {
				t.Errorf("Type = %v, want PAYOUT (negative amount = local is source)", res.Payment.Type)
			}
			if res.Payment.Metadata[MetadataKeyTransferPairID] != "12345" {
				t.Errorf("pair id = %q, want \"12345\"", res.Payment.Metadata[MetadataKeyTransferPairID])
			}
			if res.Payment.Metadata[MetadataKeyTransferDirection] != TransferDirectionOutgoing {
				t.Errorf("direction = %q, want outgoing", res.Payment.Metadata[MetadataKeyTransferDirection])
			}
			// Amount is always non-negative (PSPPayment contract).
			if res.Payment.Amount.Sign() <= 0 {
				t.Errorf("Amount must be positive, got %s", res.Payment.Amount)
			}
			if got, want := res.Payment.Amount.Cmp(big.NewInt(50_000_000)), 0; got != want {
				t.Errorf("Amount = %s, want 50_000_000 (0.5 BTC at precision 8)", res.Payment.Amount)
			}
			// Counterparty metadata absent — Bitstamp doesn't publish it.
			if _, present := res.Payment.Metadata[MetadataKeyCounterpartySubAccountID]; present {
				t.Errorf("counterparty must be omitted when not on the wire, got %+v", res.Payment.Metadata)
			}
		})
	}
}

// TestUserTransactionToPSPPayment_TransferIncoming covers the
// positive-amount branch — the local account is the destination.
func TestUserTransactionToPSPPayment_TransferIncoming(t *testing.T) {
	t.Parallel()
	tx := newTx(`{
		"id": 6789,
		"datetime": "2025-09-25 14:42:59.000000",
		"type": "14",
		"fee": "0",
		"btc": "0.50000000"
	}`)
	res, err := UserTransactionToPSPPayment(testCurrencies, tx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Payment.Type != models.PAYMENT_TYPE_PAYIN {
		t.Errorf("Type = %v, want PAYIN (positive amount = local is destination)", res.Payment.Type)
	}
	if res.Payment.Metadata[MetadataKeyTransferDirection] != TransferDirectionIncoming {
		t.Errorf("direction = %q, want incoming", res.Payment.Metadata[MetadataKeyTransferDirection])
	}
}
