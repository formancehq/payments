package tink

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/pkg/errors"
)

var (
	ErrInvalidAmount   = fmt.Errorf("invalid amount")
	ErrInvalidScale    = fmt.Errorf("invalid scale")
	ErrInvalidCurrency = fmt.Errorf("invalid currency")
)

func MapTinkAmount(unscaledValueString string, scaleString string, currencyCode string) (value *big.Int, asset *string, err error) {
	_, ok := currency.ISO4217Currencies[currencyCode]
	if !ok {
		return nil, nil, errors.Wrap(ErrInvalidCurrency, fmt.Sprintf("invalid currency code: %s", currencyCode))
	}

	unscaledValue, ok := new(big.Int).SetString(unscaledValueString, 10)
	if !ok {
		return nil, nil, errors.Wrap(ErrInvalidAmount, fmt.Sprintf("invalid amount: %s", unscaledValueString))
	}

	scale, err := strconv.Atoi(scaleString)
	if err != nil {
		return nil, nil, errors.Wrap(ErrInvalidScale, fmt.Sprintf("invalid scale: %s", scaleString))
	}

	var precision int
	if scale < 0 {
		// If scale is negative, multiply by 10^(-scale)
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-scale)), nil)
		value = new(big.Int).Mul(unscaledValue, multiplier)
		precision = 0
	} else {
		value = unscaledValue
		precision = scale
	}

	assetStr := currency.FormatAssetWithPrecision(currencyCode, precision)
	return value, &assetStr, nil
}
