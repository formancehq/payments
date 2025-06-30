package moov

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (models.CreatePayoutResponse, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return models.CreatePayoutResponse{}, err
	}

	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return models.CreatePayoutResponse{}, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %w", err),
			models.ErrInvalidRequest,
		)
	}

	sourceAccountID := models.ExtractNamespacedMetadata(pi.SourceAccount.Metadata, client.MoovAccountIDMetadataKey)
	destinationAccountID := models.ExtractNamespacedMetadata(pi.DestinationAccount.Metadata, client.MoovAccountIDMetadataKey)

	salesTaxAmount, err := extractSalesTax(pi)
	if err != nil {
		return models.CreatePayoutResponse{}, fmt.Errorf("failed to extract sales tax: %w", err)
	}

	facilitatorFee, err := extractFacilitatorFee(pi)

	if err != nil {
		return models.CreatePayoutResponse{}, fmt.Errorf("failed to extract facilitator fee: %w", err)
	}

	payoutRequest := moov.CreateTransfer{
		Source:      extractPaymentSource(pi),
		Destination: extractPaymentDestination(pi),
		Amount: moov.Amount{
			Currency: curr,
			Value:    pi.Amount.Int64(),
		},
		SalesTaxAmount: salesTaxAmount,
		Description:    pi.Description,
		FacilitatorFee: facilitatorFee,
	}

	payout, err := p.client.InitiatePayout(ctx, sourceAccountID, destinationAccountID, payoutRequest)
	if err != nil {
		return models.CreatePayoutResponse{}, err
	}

	payment, err := payoutToPayment(payout, pi.SourceAccount.Reference, pi.DestinationAccount.Reference)
	if err != nil {
		return models.CreatePayoutResponse{}, err
	}

	return models.CreatePayoutResponse{
		Payment: payment,
	}, nil
}

func payoutToPayment(
	payout *moov.Transfer,
	sourceAccountID string,
	destinationAccountID string,
) (*models.PSPPayment, error) {
	raw, err := json.Marshal(payout)
	if err != nil {
		return &models.PSPPayment{}, fmt.Errorf("failed to marshal payout: %w", err)
	}

	asset := currency.FormatAsset(supportedCurrenciesWithDecimal, payout.Amount.Currency)
	paymentType := mapPaymentType(*payout)

	metadata := mapPaymentMetadata(*payout)

	return &models.PSPPayment{
		Reference:                   payout.TransferID,
		Amount:                      big.NewInt(payout.Amount.Value),
		Asset:                       asset,
		Status:                      mapStatus(payout.Status),
		Raw:                         raw,
		Type:                        paymentType,
		CreatedAt:                   payout.CreatedOn,
		SourceAccountReference:      &sourceAccountID,
		DestinationAccountReference: &destinationAccountID,
		Metadata:                    metadata,
	}, nil
}
