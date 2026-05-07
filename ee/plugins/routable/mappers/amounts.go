package mappers

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/go-libs/v3/currency"
)

// supportedCurrencies is the ISO 4217 currency table reused across plugins.
// Routable supports a subset in practice (USD-dominant, plus FX payables);
// we let the standard helper pick precisions for any code Routable returns.
var supportedCurrencies = currency.ISO4217Currencies

// FormatAsset wraps currency.FormatAsset so callers in this package use a
// single source of truth for the supported currency table.
func FormatAsset(code string) string {
	return currency.FormatAsset(supportedCurrencies, strings.ToUpper(strings.TrimSpace(code)))
}

// PrecisionFor returns the ISO 4217 minor-unit precision for code (2 for
// USD, 0 for JPY, 3 for KWD, ...). When Routable returns an unrecognised
// code we fail loudly so the caller decides whether to skip or escalate.
func PrecisionFor(code string) (int, error) {
	c := strings.ToUpper(strings.TrimSpace(code))
	if p, ok := supportedCurrencies[c]; ok {
		return p, nil
	}
	return 0, fmt.Errorf("unsupported currency %q", code)
}

// ToMinorUnits converts a decimal Routable amount string ("100.50") to a
// *big.Int in minor units ("10050" for USD). Negative inputs are preserved.
// We round half-up at the configured precision.
func ToMinorUnits(amount string, precision int) (*big.Int, error) {
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

// FromMinorUnits is the inverse of ToMinorUnits ("10050" → "100.50" for USD).
func FromMinorUnits(amount *big.Int, precision int) string {
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

// SplitAsset splits a Formance asset string ("USD/2") into its currency
// code and precision parts. We accept the prefixed form and the bare
// currency code so PSPPaymentInitiation values from older callers work.
func SplitAsset(asset string) (string, int, error) {
	for i := 0; i < len(asset); i++ {
		if asset[i] == '/' {
			code := asset[:i]
			precision, err := PrecisionFor(code)
			if err != nil {
				return "", 0, err
			}
			return code, precision, nil
		}
	}
	precision, err := PrecisionFor(asset)
	if err != nil {
		return "", 0, err
	}
	return asset, precision, nil
}
