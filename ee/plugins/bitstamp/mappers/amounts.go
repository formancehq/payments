package mappers

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/go-libs/v3/currency"
)

func NormalizeCurrency(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// IsZeroAmount treats empty + unparseable as zero so a malformed row
// never produces a phantom non-zero amount.
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

// AbsAmount strips the leading '-'. Bitstamp signs withdrawals +
// outbound transfers negative; PSPPayment.Amount is always positive.
func AbsAmount(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "-")
}

func IsNegative(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), "-")
}

func FormatAsset(currencies map[string]int, symbol string) string {
	return currency.FormatAsset(currencies, symbol)
}

// ParseAmount converts a Bitstamp decimal string to *big.Int minor
// units. Always non-negative; callers AbsAmount first.
func ParseAmount(value string, precision int) (*big.Int, error) {
	amount, err := currency.GetAmountWithPrecisionFromString(value, precision)
	if err != nil {
		return nil, fmt.Errorf("parse amount %q at precision %d: %w", value, precision, err)
	}
	return amount, nil
}

// ResolveSinglePaymentAsset returns the single non-zero known
// currency on a user_transactions row, or ok=false when the row is
// not a single-asset payment (0 or 2+ non-zero known currencies —
// the latter is the conversion shape, handled separately).
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

// ResolveTwoAssetConversion returns the (negative, positive) leg
// pair on a conversion row. Strict: 3+ non-zero currencies or
// same-sign legs return ok=false.
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

// PrecisionFor errors on unknown currency rather than silently
// defaulting — callers decide whether to skip or escalate.
func PrecisionFor(currencies map[string]int, symbol string) (int, error) {
	p, ok := currencies[NormalizeCurrency(symbol)]
	if !ok {
		return 0, fmt.Errorf("unsupported currency %q", symbol)
	}
	return p, nil
}
