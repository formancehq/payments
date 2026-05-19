package mappers

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/go-libs/v3/currency"
)

// NormalizeCurrency canonicalises a Bitstamp currency key into the
// uppercase ticker form used everywhere as a Reference.
func NormalizeCurrency(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// IsZeroAmount reports whether a Bitstamp decimal string represents a
// zero balance. Empty strings and unparseable values are treated as
// zero so a malformed row never gets a phantom non-zero amount.
func IsZeroAmount(s string) bool {
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

// AbsAmount strips a leading minus sign. Bitstamp signs withdrawals
// and outbound transfers as negative values; PSPPayment.Amount is
// always positive per CLAUDE.md, so callers normalise here.
func AbsAmount(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "-")
}

// IsNegative reports whether a Bitstamp decimal string is signed
// negative. Used by the conversion mapper to pick the source leg
// (negative = paid out) vs the destination leg (positive = received).
func IsNegative(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), "-")
}

// FormatAsset wraps currency.FormatAsset for callers who only have the
// per-precision map.
func FormatAsset(currencies map[string]int, symbol string) string {
	return currency.FormatAsset(currencies, symbol)
}

// ParseAmount converts a Bitstamp decimal string to *big.Int minor
// units at the supplied precision. Always non-negative on return; the
// caller is expected to AbsAmount the wire value first when the sign
// is irrelevant (payments, conversion legs).
func ParseAmount(value string, precision int) (*big.Int, error) {
	amount, err := currency.GetAmountWithPrecisionFromString(value, precision)
	if err != nil {
		return nil, fmt.Errorf("parse amount %q at precision %d: %w", value, precision, err)
	}
	return amount, nil
}

// ResolveSinglePaymentAsset accepts a user_transactions CurrencyAmounts
// map and returns the single non-zero known currency, or (_, _, false,
// nil) if the row is not a single-asset payment (i.e. zero or 2+
// non-zero known currencies — the latter is the conversion shape).
//
// Returns an error only when amount parsing fails on the chosen leg;
// "no match" and "too many matches" are both non-error nil-values so
// the orchestrator skips without surfacing as a hard fail.
func ResolveSinglePaymentAsset(currencies map[string]int, amounts map[string]string) (asset string, amount *big.Int, ok bool, err error) {
	var (
		selectedSymbol  string
		selectedPrec    int
		selectedRawAbs  string
		nonZeroCount    int
	)
	for key, val := range amounts {
		symbol := NormalizeCurrency(key)
		precision, known := currencies[symbol]
		if !known {
			continue
		}
		abs := AbsAmount(val)
		if IsZeroAmount(abs) {
			continue
		}
		nonZeroCount++
		if nonZeroCount > 1 {
			return "", nil, false, nil
		}
		selectedSymbol = symbol
		selectedPrec = precision
		selectedRawAbs = abs
	}
	if nonZeroCount == 0 {
		return "", nil, false, nil
	}
	amount, err = ParseAmount(selectedRawAbs, selectedPrec)
	if err != nil {
		return "", nil, false, err
	}
	return FormatAsset(currencies, selectedSymbol), amount, true, nil
}

// TwoAssetLeg names one side of a conversion row. The two-asset
// resolver returns the negative-amount leg as Source (what the user
// paid with) and the positive-amount leg as Destination.
type TwoAssetLeg struct {
	Symbol    string
	Asset     string // "SYMBOL/<precision>"
	Precision int
	Amount    *big.Int // non-negative
}

// ResolveTwoAssetConversion accepts a user_transactions CurrencyAmounts
// map and returns the (negative, positive) currency-amount pair that
// makes up a conversion row, or (_, _, false, nil) when the row does
// not have exactly one negative + one positive known-currency amount.
//
// Detection is intentionally strict: rows with 3+ known non-zero
// currencies, two same-sign legs, or unknown currencies as one of the
// two legs all return ok=false so the orchestrator can either skip
// them or surface them at Warn.
func ResolveTwoAssetConversion(currencies map[string]int, amounts map[string]string) (source, destination TwoAssetLeg, ok bool, err error) {
	var (
		neg     *TwoAssetLeg
		pos     *TwoAssetLeg
		others  int
	)
	for key, val := range amounts {
		symbol := NormalizeCurrency(key)
		precision, known := currencies[symbol]
		if !known {
			continue
		}
		raw := strings.TrimSpace(val)
		if IsZeroAmount(AbsAmount(raw)) {
			continue
		}
		leg := TwoAssetLeg{Symbol: symbol, Precision: precision, Asset: FormatAsset(currencies, symbol)}
		switch {
		case IsNegative(raw):
			if neg != nil {
				return TwoAssetLeg{}, TwoAssetLeg{}, false, nil
			}
			amt, perr := ParseAmount(AbsAmount(raw), precision)
			if perr != nil {
				return TwoAssetLeg{}, TwoAssetLeg{}, false, perr
			}
			leg.Amount = amt
			neg = &leg
		default:
			if pos != nil {
				return TwoAssetLeg{}, TwoAssetLeg{}, false, nil
			}
			amt, perr := ParseAmount(raw, precision)
			if perr != nil {
				return TwoAssetLeg{}, TwoAssetLeg{}, false, perr
			}
			leg.Amount = amt
			pos = &leg
		}
		others++
	}
	if neg == nil || pos == nil || others != 2 {
		return TwoAssetLeg{}, TwoAssetLeg{}, false, nil
	}
	return *neg, *pos, true, nil
}

// PrecisionFor looks up a Bitstamp currency precision; returns an
// error rather than silently defaulting so the caller can decide
// whether to skip the row or escalate.
func PrecisionFor(currencies map[string]int, symbol string) (int, error) {
	p, ok := currencies[NormalizeCurrency(symbol)]
	if !ok {
		return 0, fmt.Errorf("unsupported currency %q", symbol)
	}
	return p, nil
}
