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

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequest(pi); err != nil {
		return nil, err
	}

	// Validate source and destination accounts
	if pi.SourceAccount == nil {
		return nil, models.NewConnectorValidationError("SourceAccount", fmt.Errorf("source account is required"))
	}

	if pi.DestinationAccount == nil {
		return nil, models.NewConnectorValidationError("DestinationAccount", fmt.Errorf("destination account is required"))
	}

	// Extract wallet IDs from metadata
	sourceWalletID := models.ExtractNamespacedMetadata(pi.SourceAccount.Metadata, client.MoovWalletIDMetadataKey)
	if sourceWalletID == "" {
		return nil, models.NewConnectorValidationError("SourceAccount", fmt.Errorf("source wallet ID is required"))
	}

	destinationWalletID := models.ExtractNamespacedMetadata(pi.DestinationAccount.Metadata, client.MoovWalletIDMetadataKey)
	if destinationWalletID == "" {
		return nil, models.NewConnectorValidationError("DestinationAccount", fmt.Errorf("destination wallet ID is required"))
	}

	// Get currency from asset
	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %w", err),
			models.ErrInvalidRequest,
		)
	}

	// Create transfer request
	transferReq := &client.TransferRequest{
		SourceWalletID:      sourceWalletID,
		DestinationWalletID: destinationWalletID,
		Amount:              pi.Amount.Int64(),
		Currency:            curr,
		Description:         pi.Description,
		Metadata:            pi.Metadata,
	}

	// Call the Moov API
	transfer, err := p.client.CreateTransfer(ctx, transferReq)
	if err != nil {
		return nil, err
	}

	// Convert the transfer to a payment
	payment, err := transferToPayment(transfer)
	if err != nil {
		return nil, err
	}

	return payment, nil
}