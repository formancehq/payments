package routable

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/go-libs/v3/currency"
)

// supportedCurrencies is the ISO 4217 currency table reused across plugins;
// Routable supports a subset of these in practice (USD-dominant, plus FX
// payables) but we let the standard helper pick precisions for any code
// Routable returns.
var supportedCurrencies = currency.ISO4217Currencies

// formatAsset wraps currency.FormatAsset so callers in this package use a
// single source of truth for the supported currency table.
func formatAsset(code string) string {
	return currency.FormatAsset(supportedCurrencies, strings.ToUpper(strings.TrimSpace(code)))
}

// precisionFor returns the ISO 4217 minor-unit precision for code (2 for
// USD, 0 for JPY, 3 for KWD, ...). When Routable returns an unrecognised
// code we fail loudly with a typed error so the caller can decide whether
// to skip the entity or escalate.
func precisionFor(code string) (int, error) {
	c := strings.ToUpper(strings.TrimSpace(code))
	if p, ok := supportedCurrencies[c]; ok {
		return p, nil
	}
	return 0, fmt.Errorf("unsupported currency %q", code)
}

// toMinorUnits converts a Routable decimal amount string ("100.50") to a
// *big.Int in minor units ("10050" for USD). Negative inputs are preserved.
// We round half-up at the configured precision; Routable is deliberate
// about returning amounts at the exact precision so rounding is a defensive
// move, not the common path.
func toMinorUnits(amount string, precision int) (*big.Int, error) {
	if strings.TrimSpace(amount) == "" {
		return nil, fmt.Errorf("empty amount")
	}

	rat, ok := new(big.Rat).SetString(amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount %q", amount)
	}

	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(precision)), nil)
	rat.Mul(rat, new(big.Rat).SetInt(multiplier))

	if rat.IsInt() {
		return new(big.Int).Set(rat.Num()), nil
	}

	num, denom := rat.Num(), rat.Denom()
	out := new(big.Int).Quo(num, denom)
	rem := new(big.Int).Rem(num, denom)
	rem.Abs(rem)
	doubleRem := new(big.Int).Lsh(rem, 1)
	if doubleRem.Cmp(denom) >= 0 {
		if num.Sign() >= 0 {
			out.Add(out, big.NewInt(1))
		} else {
			out.Sub(out, big.NewInt(1))
		}
	}
	return out, nil
}

// fromMinorUnits is the inverse of toMinorUnits and is used when the engine
// hands us a *big.Int amount that we have to send to Routable as a decimal
// string ("10050" → "100.50" for USD).
func fromMinorUnits(amount *big.Int, precision int) string {
	if amount == nil {
		return "0"
	}
	if precision == 0 {
		return amount.String()
	}

	negative := amount.Sign() < 0
	abs := new(big.Int).Abs(amount)
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(precision)), nil)
	quot, rem := new(big.Int).QuoRem(abs, divisor, new(big.Int))

	remStr := rem.String()
	if len(remStr) < precision {
		remStr = strings.Repeat("0", precision-len(remStr)) + remStr
	}
	out := quot.String() + "." + remStr
	if negative {
		out = "-" + out
	}
	return out
}
