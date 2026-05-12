package krakenpro

import "strings"

// krakenXZPrefixMap maps 4-character Kraken codes with X/Z prefixes to their
// standard 3-character equivalents. Only codes that actually use the legacy
// prefix convention are listed here.
var krakenXZPrefixMap = map[string]string{
	"XXBT": "XBT",
	"XETH": "ETH",
	"XLTC": "LTC",
	"XXRP": "XRP",
	"XXLM": "XLM",
	"XXDG": "DOGE",
	"XXMR": "XMR",
	"XREP": "REP",
	"XZEC": "ZEC",
	"XETC": "ETC",
	"XMLN": "MLN",
	"ZUSD": "USD",
	"ZEUR": "EUR",
	"ZGBP": "GBP",
	"ZJPY": "JPY",
	"ZCAD": "CAD",
	"ZAUD": "AUD",
}

// xbtToBTC handles the special case where XBT (ISO 4217 code for Bitcoin)
// should be normalized to BTC (common ticker).
const xbtToBTC = "XBT"

// truncateToPrecision truncates a decimal string to at most `precision` decimal places.
// Kraken returns amounts with more decimals than the currency precision (e.g. "171288.6158" for USD/2).
func truncateToPrecision(amountStr string, precision int) string {
	dotIdx := strings.IndexByte(amountStr, '.')
	if dotIdx < 0 {
		return amountStr
	}
	if precision == 0 {
		return amountStr[:dotIdx]
	}
	decimals := amountStr[dotIdx+1:]
	if len(decimals) > precision {
		decimals = decimals[:precision]
	}
	return amountStr[:dotIdx+1] + decimals
}

// normalizeAssetCode converts a Kraken asset code to a standard code.
// It handles:
// 1. Suffix stripping (.S, .F, .B, .M, .T, .P for staking/rewards variants)
// 2. X/Z prefix stripping for known 4-char codes (XXBT→XBT, ZUSD→USD)
// 3. XBT→BTC special case
func normalizeAssetCode(krakenCode string) string {
	code := strings.ToUpper(strings.TrimSpace(krakenCode))
	if code == "" {
		return code
	}

	// Strip known suffixes (.S, .F, .B, .M, .T, .P)
	for _, suffix := range []string{".S", ".F", ".B", ".M", ".T", ".P"} {
		if trimmed, found := strings.CutSuffix(code, suffix); found {
			code = trimmed
			break
		}
	}

	// Check known X/Z prefix map first
	if mapped, ok := krakenXZPrefixMap[code]; ok {
		code = mapped
	}

	// XBT → BTC
	if code == xbtToBTC {
		code = "BTC"
	}

	return code
}
