package mappers

import "testing"

func TestNormalizeAsset(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		// Legacy X/Z aliases (authoritative map, mirrors ccxt commonCurrencies).
		{"XXBT", "BTC"},
		{"XBT", "BTC"},
		{"XETH", "ETH"},
		{"ZUSD", "USD"},
		{"ZEUR", "EUR"},
		{"ZGBP", "GBP"},
		{"ZCAD", "CAD"},
		{"ZJPY", "JPY"},
		// Doge: the heuristic prefix-strip got this wrong (→"XDG"); the
		// alias map fixes it to the platform-standard DOGE.
		{"XXDG", "DOGE"},
		{"XDG", "DOGE"},
		{"XXLM", "XLM"},
		{"XXMR", "XMR"},
		{"XXRP", "XRP"},
		{"XETC", "ETC"},
		{"XLTC", "LTC"},
		{"XZEC", "ZEC"},
		// Aliases combine with staking/earn suffix stripping.
		{"XBT.M", "BTC"},
		{"XBT.F", "BTC"},
		{"XETH.S", "ETH"},
		// Plain modern tickers pass through untouched.
		{"BTC", "BTC"},
		{"USD", "USD"},
		{"ADA.S", "ADA"},
		{"ADA.M", "ADA"},
		{"BTC.B", "BTC"},
		{"BTC.F", "BTC"},
		{"BTC.P", "BTC"},
		{"BTC.T", "BTC"},
		{"EUR.HOLD", "EUR"},
		{"ADA.BASE", "ADA"},
		{"xrp", "XRP"},
		{"  btc  ", "BTC"},
		{"", ""},
		// Over-strip guard: a real 3/4-char ticker starting with X/Z that
		// is NOT a legacy-prefixed code must survive intact. The old
		// "strip any leading X/Z on a 4-char code" heuristic would have
		// mangled ZETA→ETA; the alias-map approach leaves it alone.
		{"XCN", "XCN"},
		{"ZETA", "ZETA"},
		{"ZRO", "ZRO"},
	}
	for _, c := range cases {
		got := NormalizeAsset(c.in)
		if got != c.want {
			t.Errorf("NormalizeAsset(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHasSuffixFamily(t *testing.T) {
	t.Parallel()
	for code, want := range map[string]bool{
		"XXBT":     false,
		"ADA":      false,
		"XBT.M":    true,
		"ADA.S":    true,
		"EUR.HOLD": true,
		"ada.base": true,
	} {
		if got := HasSuffixFamily(code); got != want {
			t.Errorf("HasSuffixFamily(%q) = %v want %v", code, got, want)
		}
	}
}

func TestNormalizeAssetIdempotent(t *testing.T) {
	t.Parallel()
	in := "XXBT"
	once := NormalizeAsset(in)
	twice := NormalizeAsset(once)
	if once != twice {
		t.Errorf("not idempotent: once=%q twice=%q", once, twice)
	}
}
