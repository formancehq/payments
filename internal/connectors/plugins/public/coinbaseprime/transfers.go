package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	// Wallet-to-wallet only
	if pi.SourceAccount.Metadata["spec.coinbase.com/wallet_type"] != "wallet" ||
		pi.DestinationAccount.Metadata["spec.coinbase.com/wallet_type"] != "wallet" {
		return nil, fmt.Errorf("coinbaseprime transfers currently support wallet-to-wallet only: %w", models.ErrInvalidRequest)
	}

	// Determine currency symbol and human-readable amount for SDK
	symbol, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed to get currency and precision from asset: %v: %w", err, models.ErrInvalidRequest)
	}
	amountStr, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return nil, fmt.Errorf("failed to get string amount from big int: %v: %w", err, models.ErrInvalidRequest)
	}

	portfolioID := pi.SourceAccount.Metadata["spec.coinbase.com/portfolio_id"]
	idempotencyKey := models.IdempotencyKey(pi.Reference)

	resp, err := p.client.InitiateTransfer(
		ctx,
		&client.TransferRequest{
			PortfolioID:         portfolioID,
			WalletID:            pi.SourceAccount.Reference,
			DestinationWalletID: pi.DestinationAccount.Reference,
			Amount:              amountStr,
			CurrencySymbol:      symbol,
			IdempotencyKey:      idempotencyKey,
		},
	)
	if err != nil {
		return nil, err
	}

	return transferToPayment(resp)
}

func transferToPayment(transfer *client.TransferResponse) (*models.PSPPayment, error) {
	raw := transfer.Raw
	precision, ok := supportedCurrenciesWithDecimal[transfer.Symbol]
	if !ok {
		precision = 8
	}
	amount, err := currency.GetAmountWithPrecisionFromString(transfer.Amount, precision)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount: %w", err)
	}
	asset := currency.FormatAsset(supportedCurrenciesWithDecimal, transfer.Symbol)
	if asset == "" {
		asset = fmt.Sprintf("%s/%d", transfer.Symbol, precision)
	}

	if raw == nil {
		raw, _ = json.Marshal(transfer)
	}

	reference := transfer.ID
	if reference == "" {
		reference = transfer.IdempotencyKey
	}

	return &models.PSPPayment{
		Reference:                   reference,
		CreatedAt:                   time.Now().UTC(),
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Amount:                      amount,
		Asset:                       asset,
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      models.PAYMENT_STATUS_PENDING,
		SourceAccountReference:      &transfer.FromWalletID,
		DestinationAccountReference: &transfer.ToWalletID,
		Raw:                         raw,
	}, nil
}
