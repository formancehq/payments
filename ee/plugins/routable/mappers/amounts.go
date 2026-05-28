package mappers

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
)

var supportedCurrencies = currency.ISO4217Currencies

func FormatAsset(code string) string {
	return currency.FormatAsset(supportedCurrencies, strings.ToUpper(strings.TrimSpace(code)))
}

// PrecisionFor errors on unrecognised codes so callers can decide
// whether to skip the row or escalate, rather than silently defaulting.
func PrecisionFor(code string) (int, error) {
	c := strings.ToUpper(strings.TrimSpace(code))
	if p, ok := supportedCurrencies[c]; ok {
		return p, nil
	}
	return 0, fmt.Errorf("unsupported currency %q", code)
}

// ToMinorUnits parses a decimal string ("100.50") to *big.Int minor units.
// Half-up rounding is defensive — Routable returns amounts at the
// configured precision in practice.
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

// SplitAsset accepts both "USD/2" and bare "USD" so older callers keep
// working.
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
