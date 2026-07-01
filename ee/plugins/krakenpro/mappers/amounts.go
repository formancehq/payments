package mappers

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
)

// FormatAsset returns Formance's canonical "<SYMBOL>/<precision>" form
// using the cached currencies map populated from /0/public/Assets.
func FormatAsset(currencies map[string]int, symbol string) string {
	return currency.FormatAsset(currencies, symbol)
}

// IsZeroAmount reports whether s is an empty or parseable-zero amount.
// A malformed value returns false (treated as non-zero) so callers that
// gate on it don't silently skip the row — the subsequent parse surfaces
// the error instead.
func IsZeroAmount(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return true
	}
	f, _, err := new(big.Float).Parse(s, 10)
	if err != nil {
		return false
	}
	return f.Sign() == 0
}

// IsNegative reports whether the value starts with '-'.
func IsNegative(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), "-")
}

// AbsAmount strips the leading '-'. Kraken signs withdrawals + outbound
// transfers negative; PSPPayment.Amount is always positive.
func AbsAmount(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "-")
}

// ParseDecimalAmount converts a Kraken decimal string into *big.Int
// minor units at the supplied precision. Excess fractional digits are
// truncated rather than rejected — Kraken's `display_decimals` is
// often less than the precision returned in raw ledger rows, and
// the currency-config `decimals` field is the authoritative bound.
func ParseDecimalAmount(value string, precision int) (*big.Int, error) {
	if value == "" {
		return nil, fmt.Errorf("empty amount string")
	}
	amount, err := currency.GetAmountWithPrecisionFromString(value, precision)
	if err != nil {
		if errors.Is(err, currency.ErrInvalidPrecision) && precision > 0 {
			if idx := strings.IndexByte(value, '.'); idx >= 0 {
				decimalPart := value[idx+1:]
				if len(decimalPart) > precision {
					truncated := value[:idx+1+precision]
					amount, err = currency.GetAmountWithPrecisionFromString(truncated, precision)
					if err == nil {
						return amount, nil
					}
				}
			}
		}
		return nil, fmt.Errorf("parse decimal %q at precision %d: %w", value, precision, err)
	}
	return amount, nil
}

// SubDecimal returns lhs - rhs using *big.Int arithmetic. Both inputs
// are decimal strings; precision is applied to both. Negative results
// are clamped to zero — `hold_trade` is never expected to exceed the
// gross `balance`, but a transient overshoot mustn't crash the cycle.
func SubDecimal(lhs, rhs string, precision int) (*big.Int, error) {
	l, err := ParseDecimalAmount(orZero(lhs), precision)
	if err != nil {
		return nil, fmt.Errorf("lhs: %w", err)
	}
	r, err := ParseDecimalAmount(orZero(rhs), precision)
	if err != nil {
		return nil, fmt.Errorf("rhs: %w", err)
	}
	res := new(big.Int).Sub(l, r)
	if res.Sign() < 0 {
		res.SetInt64(0)
	}
	return res, nil
}

// AvailableBalance computes Kraken's spendable balance per the BalanceEx
// docs: balance + credit - credit_used - hold_trade, clamped to >= 0.
//   - balance:     settled funds held in the asset class
//   - credit:      a VIP/Pro credit line extended on top of balance
//   - credit_used: the portion of that credit line already drawn
//   - hold_trade:  funds locked behind open orders
//
// credit / credit_used are empty (zero) on spot-only accounts and on
// VIP/Pro accounts without a credit line, so the formula collapses to
// balance - hold_trade there. The clamp guards a transient hold_trade
// overshoot from producing a negative (invalid) balance.
func AvailableBalance(balance, holdTrade, credit, creditUsed string, precision int) (*big.Int, error) {
	terms := []struct {
		s    string
		sign int
	}{
		{balance, +1},
		{credit, +1},
		{creditUsed, -1},
		{holdTrade, -1},
	}
	total := new(big.Int)
	for _, t := range terms {
		v, err := ParseDecimalAmount(orZero(t.s), precision)
		if err != nil {
			return nil, err
		}
		if t.sign < 0 {
			total.Sub(total, v)
		} else {
			total.Add(total, v)
		}
	}
	if total.Sign() < 0 {
		total.SetInt64(0)
	}
	return total, nil
}

func orZero(s string) string {
	if strings.TrimSpace(s) == "" {
		return "0"
	}
	return s
}
