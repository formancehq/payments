package mappers

import (
	"math/big"
	"testing"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/models"
)

func TestLedgerEntryToPSPPaymentDeposit(t *testing.T) {
	t.Parallel()
	entry := client.LedgerEntry{
		Refid:   "TYH2WW-WHIOM-TFFLE6",
		Time:    1688019200.5,
		Type:    "deposit",
		Asset:   "ZEUR",
		Amount:  "100.00",
		Balance: "1234.56",
	}
	res, err := LedgerEntryToPSPPayment(testCurrencies, testWallets, "L4UESK-KG3EQ-UFO4T5", entry)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Skip || res.Payment == nil {
		t.Fatal("expected payment emission")
	}
	p := res.Payment
	if p.Reference != "L4UESK-KG3EQ-UFO4T5" {
		t.Errorf("reference=%q", p.Reference)
	}
	if p.Type != models.PAYMENT_TYPE_PAYIN {
		t.Errorf("type=%v want PAYIN", p.Type)
	}
	if p.Asset != "EUR/2" {
		t.Errorf("asset=%q want EUR/2", p.Asset)
	}
	want := big.NewInt(10000)
	if p.Amount.Cmp(want) != 0 {
		t.Errorf("amount=%s want %s", p.Amount, want)
	}
	if p.Status != models.PAYMENT_STATUS_SUCCEEDED {
		t.Errorf("status=%v", p.Status)
	}
	// PAYIN credits the destination (the asset's spot account).
	if p.DestinationAccountReference == nil || *p.DestinationAccountReference != testWallets["EUR"] {
		t.Errorf("dest ref=%v want %q", p.DestinationAccountReference, testWallets["EUR"])
	}
	if p.SourceAccountReference != nil {
		t.Errorf("PAYIN should leave source ref nil, got %v", *p.SourceAccountReference)
	}
}

func TestLedgerEntryToPSPPaymentWithdrawal(t *testing.T) {
	t.Parallel()
	entry := client.LedgerEntry{
		Type: "withdrawal", Asset: "XXBT", Amount: "-0.5", Time: 1.0,
	}
	res, err := LedgerEntryToPSPPayment(testCurrencies, testWallets, "L1", entry)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Payment == nil {
		t.Fatal("expected emission")
	}
	if res.Payment.Type != models.PAYMENT_TYPE_PAYOUT {
		t.Errorf("type=%v", res.Payment.Type)
	}
	if res.Payment.Amount.Sign() <= 0 {
		t.Errorf("amount should be positive: %s", res.Payment.Amount)
	}
	// PAYOUT debits the source (the asset's spot account).
	if res.Payment.SourceAccountReference == nil || *res.Payment.SourceAccountReference != testWallets["BTC"] {
		t.Errorf("source ref=%v want %q", res.Payment.SourceAccountReference, testWallets["BTC"])
	}
	if res.Payment.DestinationAccountReference != nil {
		t.Errorf("PAYOUT should leave dest ref nil, got %v", *res.Payment.DestinationAccountReference)
	}
}

func TestLedgerEntryToPSPPaymentTransferIsTransfer(t *testing.T) {
	t.Parallel()
	// A transfer is an internal movement: always TRANSFER, with the spot
	// (known) leg attributed by amount sign — negative leaves the account
	// (source), positive enters it (destination).
	out, err := LedgerEntryToPSPPayment(testCurrencies, testWallets, "L1", client.LedgerEntry{
		Type: "transfer", Asset: "ZUSD", Amount: "-50.00", Time: 1.0,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out.Payment.Type != models.PAYMENT_TYPE_TRANSFER {
		t.Errorf("transfer should map to TRANSFER, got %v", out.Payment.Type)
	}
	if out.Payment.SourceAccountReference == nil {
		t.Error("negative transfer should attribute the spot account as source")
	}
	if out.Payment.DestinationAccountReference != nil {
		t.Error("negative transfer should leave destination nil")
	}

	in, err := LedgerEntryToPSPPayment(testCurrencies, testWallets, "L2", client.LedgerEntry{
		Type: "transfer", Asset: "ZUSD", Amount: "50.00", Time: 1.0,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if in.Payment.Type != models.PAYMENT_TYPE_TRANSFER {
		t.Errorf("transfer should map to TRANSFER, got %v", in.Payment.Type)
	}
	if in.Payment.DestinationAccountReference == nil {
		t.Error("positive transfer should attribute the spot account as destination")
	}
	if in.Payment.SourceAccountReference != nil {
		t.Error("positive transfer should leave source nil")
	}
}

func TestLedgerEntryToPSPPaymentSkipsTrade(t *testing.T) {
	t.Parallel()
	res, err := LedgerEntryToPSPPayment(testCurrencies, testWallets, "L1", client.LedgerEntry{
		Type: "trade", Asset: "XXBT", Amount: "-0.1", Time: 1.0,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.Skip {
		t.Error("trade row should be skipped")
	}
}

func TestLedgerEntryToPSPPaymentSkipsConversion(t *testing.T) {
	t.Parallel()
	res, err := LedgerEntryToPSPPayment(testCurrencies, testWallets, "L1", client.LedgerEntry{
		Type: "conversion", Asset: "ZUSD", Amount: "-10.00", Time: 1.0,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.Skip {
		t.Error("conversion row should be skipped")
	}
}

func TestLedgerEntryToPSPPaymentUnknownTypeWarns(t *testing.T) {
	t.Parallel()
	res, err := LedgerEntryToPSPPayment(testCurrencies, testWallets, "L1", client.LedgerEntry{
		Type: "future_unknown", Asset: "ZUSD", Amount: "10.00", Time: 1.0,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Payment == nil {
		t.Fatal("expected emission as OTHER")
	}
	if res.Payment.Type != models.PAYMENT_TYPE_OTHER {
		t.Errorf("type=%v want OTHER", res.Payment.Type)
	}
	if !res.UnknownType {
		t.Error("expected UnknownType=true to drive warn-log")
	}
}

func TestLedgerEntryToPSPPaymentUnknownAssetSkipped(t *testing.T) {
	t.Parallel()
	res, err := LedgerEntryToPSPPayment(testCurrencies, testWallets, "L1", client.LedgerEntry{
		Type: "deposit", Asset: "XYZ", Amount: "1.0", Time: 1.0,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.Skip {
		t.Error("unknown asset should be skipped")
	}
}
