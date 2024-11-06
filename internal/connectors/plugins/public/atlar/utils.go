package atlar

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func ParseAtlarTimestamp(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}

func validateTransferPayoutRequest(pi models.PSPPaymentInitiation) error {
	if pi.SourceAccount == nil {
		return fmt.Errorf("source account is required: %w", models.ErrInvalidRequest)
	}

	if pi.DestinationAccount == nil {
		return fmt.Errorf("destination account is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func amountToString(amount big.Int, precision int) string {
	raw := amount.String()
	if precision < 0 {
		precision = 0
	}
	insertPosition := len(raw) - precision
	if insertPosition <= 0 {
		return "0." + strings.Repeat("0", -insertPosition) + raw
	}
	return raw[:insertPosition] + "." + raw[insertPosition:]
}
