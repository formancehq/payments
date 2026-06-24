package client

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/pkg/domain/plugins"
)

func TranslatePlaidAmount(
	amount float64,
	currencyCode string,
) (*big.Int, string, error) {
	precision, err := currency.GetPrecision(currency.ISO4217Currencies, currencyCode)
	if err != nil {
		if errors.Is(err, currency.ErrMissingCurrencies) {
			// Wrap as ErrCurrencyNotSupported so callers can skip the record
			// instead of failing (a retryable error freezes ingestion).
			return nil, "", fmt.Errorf("%w: %w", plugins.ErrCurrencyNotSupported, err)
		}
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
