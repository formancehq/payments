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
	standardPrecision, ok := currency.ISO4217Currencies[currencyCode]
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

	scaleDifference := standardPrecision - scale
	if scaleDifference > 0 {
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scaleDifference)), nil)
		value = new(big.Int).Mul(unscaledValue, multiplier)
	} else if scaleDifference < 0 {
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-scaleDifference)), nil)
		value = new(big.Int).Div(unscaledValue, divisor)
	} else {
		value = unscaledValue
	}

	assetStr := currency.FormatAssetWithPrecision(currencyCode, standardPrecision)
	return value, &assetStr, nil
}
