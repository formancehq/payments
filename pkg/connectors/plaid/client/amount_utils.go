package client

import (
	"math/big"
	"strconv"

	"github.com/formancehq/go-libs/v3/currency"
)

func TranslatePlaidAmount(
	amount float64,
	currencyCode string,
) (*big.Int, string, error) {
	precision, err := currency.GetPrecision(currency.ISO4217Currencies, currencyCode)
	if err != nil {
		return nil, "", err
	}

	amountString := strconv.FormatFloat(amount, 'f', -1, 64)
	amountInt, err := currency.GetAmountWithPrecisionFromString(amountString, precision)
	if err != nil {
		return nil, "", err
	}

	assetName := currency.FormatAssetWithPrecision(currencyCode, precision)

	return amountInt, assetName, nil
}
