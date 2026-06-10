package mappers

import "strings"

// MetadataPrefix namespaces every metadata key the connector emits.
const MetadataPrefix = "com.krakenpro.spec/"

// suffixFamilies lists the documented Kraken asset-code suffix
// families. We fold all of them onto the base symbol so the same
// underlying asset doesn't fan out into multiple PSPAccounts.
//
// Sources:
//   - .S: legacy staked (https://support.kraken.com/articles/staking-products)
//   - .M: opt-in rewards
//   - .B: yield-bearing
//   - .F: Kraken Rewards (auto-earn)
//   - .P: parachain staked
//   - .T: tokenised
//   - .HOLD: pending hold
//   - .BASE: margin base
var suffixFamilies = []string{".S", ".M", ".B", ".F", ".P", ".T", ".HOLD", ".BASE"}

// assetAliases maps Kraken's legacy X/Z-class codes to platform symbols
// (mirrors ccxt's commonCurrencies). An explicit allowlist, not a blind
// leading-X/Z strip, because some codes don't strip cleanly (XXDG is
// DOGE, not "XDG") and a blind strip mangles real tickers that start
// with X/Z (XCN, ZETA, ZRO). The legacy set is closed; unmapped codes
// pass through unchanged (still consistent — /Assets keys normalize the
// same way).
var assetAliases = map[string]string{
	"XXBT": "BTC", "XBT": "BTC",
	"XXDG": "DOGE", "XDG": "DOGE",
	"XETH": "ETH",
	"XXRP": "XRP",
	"XXLM": "XLM",
	"XXMR": "XMR",
	"XLTC": "LTC",
	"XETC": "ETC",
	"XZEC": "ZEC",
	"XMLN": "MLN",
	"XREP": "REP",
	"ZUSD": "USD", "ZEUR": "EUR", "ZGBP": "GBP",
	"ZCAD": "CAD", "ZJPY": "JPY", "ZAUD": "AUD",
}

// NormalizeAsset returns the canonical Formance-side symbol for a
// Kraken asset code. It is idempotent and case-insensitive.
//
//	XXBT      → BTC    (legacy alias)
//	XXDG      → DOGE   (alias — heuristic prefix-strip got this wrong)
//	ZUSD      → USD    (legacy fiat alias)
//	XBT.M     → BTC    (strip earn suffix, then alias)
//	ADA.S     → ADA    (strip suffix family)
//	ZETA      → ZETA   (not a legacy code — left intact)
//	BTC       → BTC
//
// Empty / whitespace-only inputs return "".
func NormalizeAsset(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return ""
	}
	for _, suffix := range suffixFamilies {
		if strings.HasSuffix(code, suffix) {
			code = strings.TrimSuffix(code, suffix)
			break
		}
	}
	if canonical, ok := assetAliases[code]; ok {
		return canonical
	}
	return code
}

// HasSuffixFamily reports whether a raw Kraken code carries a
// staking/earn suffix (.S/.M/.F/…) — i.e. it is not the spot code.
func HasSuffixFamily(code string) bool {
	code = strings.ToUpper(code)
	for _, suffix := range suffixFamilies {
		if strings.HasSuffix(code, suffix) {
			return true
		}
	}
	return false
}
