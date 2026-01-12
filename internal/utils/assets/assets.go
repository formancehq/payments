package assets

import (
	"regexp"
)

// Pattern validates asset symbols:
// - Must start with uppercase letter [A-Z]
// - Followed by 0-16 uppercase letters, digits, or underscores [A-Z0-9_]
// - Optional precision suffix: slash followed by 1-6 digits (e.g., /8)
// Examples: BTC, ETH, USD, BTC_TEST, USDC_ETH, BTC/8, ETH_TEST5/18
const Pattern = `[A-Z][A-Z0-9_]{0,16}(\/\d{1,6})?`

var Regexp = regexp.MustCompile("^" + Pattern + "$")

func IsValid(v string) bool {
	return Regexp.Match([]byte(v))
}
