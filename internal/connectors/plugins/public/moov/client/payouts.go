package client

import (
	"context"
	"encoding/json"
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
		PartnerAccountID:     c.accountID,
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

	// Create the transfer with the selected payment methods
	transfer := moov.CreateTransfer{
		Amount: moov.Amount{
			Currency: pr.Amount.Currency,
			Value:    pr.Amount.Value,
		},
		Source:         pr.Source,
		Destination:    pr.Destination,
		SalesTaxAmount: pr.SalesTaxAmount,
		FacilitatorFee: pr.FacilitatorFee,
		Description:    pr.Description,
	}

	transferData, _ := json.Marshal(transfer)
	fmt.Println("___TRANSFER_DATA___", string(transferData))

	payout, _, err := c.service.CreateMoovTransfer(ctx, c.accountID, transfer)
	if err != nil {
		return nil, models.NewConnectorValidationError("", fmt.Errorf("failed to create payout: %w", err))
	}

	return payout, nil
}
