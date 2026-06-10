package mappers

import (
	"testing"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
)

func TestLedgerMetadata(t *testing.T) {
	t.Parallel()
	t.Run("all-fields-present", func(t *testing.T) {
		t.Parallel()
		m := LedgerMetadata("L-1", client.LedgerEntry{
			Refid: "REF", Type: "deposit", Aclass: "currency", Asset: "XXBT",
			Subtype: "spot", Fee: "0.5", Balance: "1.234",
		})
		mustHave(t, m, MetadataPrefix+"ledger_id", "L-1")
		mustHave(t, m, MetadataPrefix+"refid", "REF")
		mustHave(t, m, MetadataPrefix+"kraken_type", "deposit")
		mustHave(t, m, MetadataPrefix+"kraken_asset", "XXBT")
		mustHave(t, m, MetadataPrefix+"aclass", "currency")
		mustHave(t, m, MetadataPrefix+"subtype", "spot")
		mustHave(t, m, MetadataPrefix+"fee", "0.5")
		mustHave(t, m, MetadataPrefix+"balance_after", "1.234")
	})
	t.Run("empty-subtype-omitted", func(t *testing.T) {
		t.Parallel()
		m := LedgerMetadata("L-1", client.LedgerEntry{Type: "deposit"})
		if _, ok := m[MetadataPrefix+"subtype"]; ok {
			t.Fatal("empty subtype must be omitted")
		}
	})
	t.Run("zero-fee-omitted", func(t *testing.T) {
		t.Parallel()
		m := LedgerMetadata("L-1", client.LedgerEntry{Type: "deposit", Fee: "0.00000000"})
		if _, ok := m[MetadataPrefix+"fee"]; ok {
			t.Fatal("zero fee must be omitted")
		}
	})
}

func TestAccountMetadata(t *testing.T) {
	t.Parallel()
	t.Run("spot", func(t *testing.T) {
		t.Parallel()
		m := AccountMetadata("XXBT")
		mustHave(t, m, MetadataPrefix+"wallet_type", WalletClassSpot)
		mustHave(t, m, MetadataPrefix+"kraken_asset", "XXBT")
	})
	t.Run("earn-variant", func(t *testing.T) {
		t.Parallel()
		m := AccountMetadata("xbt.m")
		mustHave(t, m, MetadataPrefix+"wallet_type", "rewards")
		mustHave(t, m, MetadataPrefix+"kraken_asset", "XBT.M")
	})
	if WalletClassSpot != "spot" {
		t.Fatalf("WalletClassSpot drifted: %q", WalletClassSpot)
	}
}

func TestWalletClass(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"XXBT":     "spot",
		"ADA":      "spot",
		"ADA.S":    "staked",
		"XBT.M":    "rewards",
		"BTC.B":    "yield",
		"XBT.F":    "earn",
		"DOT.P":    "parachain",
		"FOO.T":    "tokenised",
		"EUR.HOLD": "hold",
		"ADA.BASE": "margin",
	}
	for code, want := range cases {
		if got := WalletClass(code); got != want {
			t.Errorf("WalletClass(%q) = %q want %q", code, got, want)
		}
	}
}

func TestOrderMetadata(t *testing.T) {
	t.Parallel()
	t.Run("with-fills", func(t *testing.T) {
		t.Parallel()
		m := OrderMetadata("XXBTZUSD", "XBT/USD", []string{"T-1", "T-2"}, "limit", "USD/2")
		mustHave(t, m, MetadataPrefix+"pair", "XXBTZUSD")
		mustHave(t, m, MetadataPrefix+"ws_name", "XBT/USD")
		mustHave(t, m, MetadataPrefix+"ordertype", "limit")
		mustHave(t, m, MetadataPrefix+"price_asset", "USD/2")
		mustHave(t, m, MetadataPrefix+"fills", "T-1,T-2")
	})
	t.Run("no-fills-omitted", func(t *testing.T) {
		t.Parallel()
		m := OrderMetadata("XXBTZUSD", "XBT/USD", nil, "market", "USD/2")
		if _, ok := m[MetadataPrefix+"fills"]; ok {
			t.Fatal("zero fills → key must be omitted")
		}
	})
}

func mustHave(t *testing.T, m map[string]string, k, want string) {
	t.Helper()
	got, ok := m[k]
	if !ok {
		t.Fatalf("missing key %q", k)
	}
	if got != want {
		t.Fatalf("key %q: got %q want %q", k, got, want)
	}
}
