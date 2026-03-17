package bitstamp

import (
	"math/big"
	"strings"
)

// normalizeCurrency uppercases and trims a currency symbol.
func normalizeCurrency(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// isZeroAmount returns true if the string represents a zero or empty amount.
// Parses as a decimal to handle any precision (e.g. "0.00", "0.00000000").
func isZeroAmount(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return true
	}
	f, _, err := new(big.Float).Parse(s, 10)
	if err != nil {
		return true
	}
	return f.Sign() == 0
}
