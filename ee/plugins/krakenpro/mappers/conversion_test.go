package mappers

import (
	"math/big"
	"testing"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

func TestPairConversionLegs(t *testing.T) {
	t.Parallel()
	neg := ConversionLeg{LedgerID: "L1", Entry: client.LedgerEntry{Asset: "ZUSD", Amount: "-100.00"}}
	pos := ConversionLeg{LedgerID: "L2", Entry: client.LedgerEntry{Asset: "XXBT", Amount: "0.0036"}}

	src, dst, ok := PairConversionLegs(pos, neg) // arbitrary order
	if !ok {
		t.Fatal("expected pair to resolve")
	}
	if src.LedgerID != "L1" || dst.LedgerID != "L2" {
		t.Errorf("expected source=L1 (neg), dest=L2 (pos); got src=%s dst=%s", src.LedgerID, dst.LedgerID)
	}
}

func TestPairConversionLegsSameSign(t *testing.T) {
	t.Parallel()
	a := ConversionLeg{LedgerID: "L1", Entry: client.LedgerEntry{Asset: "BTC", Amount: "0.01"}}
	b := ConversionLeg{LedgerID: "L2", Entry: client.LedgerEntry{Asset: "ETH", Amount: "0.1"}}
	if _, _, ok := PairConversionLegs(a, b); ok {
		t.Error("same-sign pair must not resolve")
	}
}

func TestConversionPairToPSPConversion(t *testing.T) {
	t.Parallel()
	src := ConversionLeg{LedgerID: "L1", Entry: client.LedgerEntry{
		Refid: "C1", Type: "conversion", Asset: "ZUSD", Amount: "-100.00", Time: 1.0,
	}}
	dst := ConversionLeg{LedgerID: "L2", Entry: client.LedgerEntry{
		Refid: "C1", Type: "conversion", Asset: "XXBT", Amount: "0.0036", Fee: "0.0001", Time: 2.0,
	}}
	got, err := ConversionPairToPSPConversion(testCurrencies, testWallets, src, dst)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Reference != "C1" {
		t.Errorf("reference=%q", got.Reference)
	}
	if got.SourceAsset != "USD/2" || got.DestinationAsset != "BTC/8" {
		t.Errorf("assets=%s→%s", got.SourceAsset, got.DestinationAsset)
	}
	// Account references resolve to the spot account of each leg's symbol.
	if got.SourceAccountReference == nil || *got.SourceAccountReference != testWallets["USD"] {
		t.Errorf("source ref=%v want %q", got.SourceAccountReference, testWallets["USD"])
	}
	if got.DestinationAccountReference == nil || *got.DestinationAccountReference != testWallets["BTC"] {
		t.Errorf("dest ref=%v want %q", got.DestinationAccountReference, testWallets["BTC"])
	}
	if got.SourceAmount.Cmp(big.NewInt(10000)) != 0 {
		t.Errorf("source amount=%s", got.SourceAmount)
	}
	if got.Status != models.CONVERSION_STATUS_COMPLETED {
		t.Errorf("status=%v", got.Status)
	}
	if got.Fee == nil || got.Fee.Sign() <= 0 {
		t.Error("expected non-zero destination fee")
	}
}

func TestConversionPairToPSPConversion_SourceFeeMetadataOnly(t *testing.T) {
	t.Parallel()
	src := ConversionLeg{LedgerID: "L1", Entry: client.LedgerEntry{
		Refid: "C1", Type: "conversion", Asset: "ZUSD", Amount: "-100.00", Fee: "0.50", Time: 1,
	}}
	dst := ConversionLeg{LedgerID: "L2", Entry: client.LedgerEntry{
		Refid: "C1", Type: "conversion", Asset: "XXBT", Amount: "0.0036", Time: 1,
	}}
	got, err := ConversionPairToPSPConversion(testCurrencies, testWallets, src, dst)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Metadata[MetadataPrefix+"source_fee"] != "0.50" {
		t.Errorf("source_fee metadata missing: %v", got.Metadata)
	}
	if got.Fee != nil {
		t.Error("destination Fee should remain nil when only source has a fee")
	}
}

func TestConversionPairToPSPConversion_UnknownAsset(t *testing.T) {
	t.Parallel()
	src := ConversionLeg{LedgerID: "L1", Entry: client.LedgerEntry{Refid: "C1", Asset: "ZZZ", Amount: "-1"}}
	dst := ConversionLeg{LedgerID: "L2", Entry: client.LedgerEntry{Refid: "C1", Asset: "XXBT", Amount: "1"}}
	if _, err := ConversionPairToPSPConversion(testCurrencies, testWallets, src, dst); err == nil {
		t.Fatal("expected unknown-source-asset error")
	}
	src.Entry.Asset, dst.Entry.Asset = "XXBT", "ZZZ"
	if _, err := ConversionPairToPSPConversion(testCurrencies, testWallets, src, dst); err == nil {
		t.Fatal("expected unknown-destination-asset error")
	}
}
