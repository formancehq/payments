package mappers

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/formancehq/go-libs/v3/currency"
)

func ParseDecimalAmount(value string, precision int) (*big.Int, error) {
	if value == "" {
		return nil, fmt.Errorf("empty amount string")
	}
	amount, err := currency.GetAmountWithPrecisionFromString(value, precision)
	if err != nil {

		// Bitstamp often gives us amounts with more decimals than their currency endpoint tells us the precision is
		// if the trailing numbers are 00s we will truncate them and try to proceed anyway
		if errors.Is(err, currency.ErrInvalidPrecision) && precision > 0 {
			if idx := strings.IndexByte(value, '.'); idx >= 0 {
				decimalPart := value[idx+1:]
				if len(decimalPart) > precision && strings.TrimLeft(decimalPart[precision:], "0") == "" {
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
