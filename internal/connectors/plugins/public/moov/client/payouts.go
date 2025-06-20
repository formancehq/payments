package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

func (c *client) InitiatePayout(
	ctx context.Context,
	sourceAccountID string,
	destinationAccountID string,
	pr moov.CreateTransfer) (*moov.Transfer, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_payout")

	/**
	 * Get transfer options from Moov
	 *
	 * This will return the available payment methods for the source and destination accounts
	 * We can only use the payment methods that are returned in the options,
	 *
	 */
	paymentData := PaymentOptionsRequest{
		SourceAccountID:      sourceAccountID,
		DestinationAccountID: destinationAccountID,
		Amount:               pr.Amount.Value,
		Currency:             pr.Amount.Currency,
	}

	transferOptions, err := c.service.GetMoovTransferOptions(ctx, paymentData)
	if err != nil {
		return nil, models.NewConnectorValidationError("paymentOptions", fmt.Errorf("failed to get transfer options: %w", err))
	}

	if len(transferOptions.SourceOptions) == 0 {
		return nil, models.NewConnectorValidationError("SourceAccountID", errors.New("no source options found in Moov for source account"))
	}

	if len(transferOptions.DestinationOptions) == 0 {
		return nil, models.NewConnectorValidationError("DestinationAccountID", errors.New("no destination options found in Moov for destination account"))
	}

	var sourcePaymentMethod *moov.PaymentMethod
	var destinationPaymentMethod *moov.PaymentMethod

	// Check if source payment method exists in the transfer options
	for _, option := range transferOptions.SourceOptions {
		if option.PaymentMethodID == pr.Source.PaymentMethodID {
			sourcePaymentMethod = &option
			break
		}
	}

	if sourcePaymentMethod == nil {
		return nil, models.NewConnectorValidationError("SourceAccountID", fmt.Errorf("source payment method %s not found in available transfer options", pr.Source.PaymentMethodID))
	}

	// Check if destination payment method exists in the transfer options

	for _, option := range transferOptions.DestinationOptions {
		if option.PaymentMethodID == pr.Destination.PaymentMethodID {
			destinationPaymentMethod = &option
			break
		}
	}

	if destinationPaymentMethod == nil {
		return nil, models.NewConnectorValidationError("DestinationAccountID", fmt.Errorf("destination payment method %s not found in available transfer options", pr.Destination.PaymentMethodID))
	}

	// Validate transaction amount against source payment method limit
	if sourcePaymentMethod.PaymentMethodType != "" {
		sourceMethodType := PaymentMethodType(sourcePaymentMethod.PaymentMethodType)
		isValid, limit := ValidateTransactionLimit(sourceMethodType, float64(pr.Amount.Value))
		if !isValid {
			return nil, models.NewConnectorValidationError("amount", fmt.Errorf("transaction amount %d exceeds source payment method limit of %f for %s",
				pr.Amount.Value, limit, sourceMethodType))
		}
	}

	// Validate transaction amount against destination payment method limit
	if destinationPaymentMethod.PaymentMethodType != "" {
		destMethodType := PaymentMethodType(destinationPaymentMethod.PaymentMethodType)
		isValid, limit := ValidateTransactionLimit(destMethodType, float64(pr.Amount.Value))
		if !isValid {
			return nil, models.NewConnectorValidationError("amount", fmt.Errorf("transaction amount %d exceeds destination payment method limit of %f for %s",
				pr.Amount.Value, limit, destMethodType))
		}
	}

	// Create the transfer with the selected payment methods
	transfer := moov.CreateTransfer{
		Amount: moov.Amount{
			Currency: pr.Amount.Currency,
			Value:    pr.Amount.Value,
		},
		Source: moov.CreateTransfer_Source{
			PaymentMethodID: pr.Source.PaymentMethodID,
		},
		Destination: moov.CreateTransfer_Destination{
			PaymentMethodID: pr.Destination.PaymentMethodID,
		},
	}

	payout, _, err := c.service.CreateMoovTransfer(ctx, c.accountID, transfer)
	if err != nil {
		return nil, models.NewConnectorValidationError("", fmt.Errorf("failed to create payout: %w", err))
	}

	return payout, nil
}
