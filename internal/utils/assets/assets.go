package assets

import (
	"regexp"
)

// Pattern allows uppercase letters, digits, and underscores for asset identifiers
// (underscores are common in crypto assets like BTC_TEST, ETH_TEST5, etc.)
const Pattern = `[A-Z][A-Z0-9_]{0,16}(\/\d{1,6})?`

var Regexp = regexp.MustCompile("^" + Pattern + "$")

func IsValid(v string) bool {
	return Regexp.Match([]byte(v))
}
