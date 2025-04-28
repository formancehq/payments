package qonto

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	// TODO: Since we are in minor units currency in the PSPPaymentInitiation
	// object, we sometimes need to put back the amount in float for
	// the PSP. You can use the next methods to do that.
	// curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	// if err != nil {
	//	 return nil, fmt.Errorf("failed to get currency and precision from asset: %v: %w", err, models.ErrInvalidRequest)
	// }

	// amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	// if err != nil {
	// 	 return nil, fmt.Errorf("failed to get string amount from big int: %v: %w", err, models.ErrInvalidRequest)
	// }

	resp, err := p.client.InitiateTransfer(
		ctx,
		&client.TransferRequest{
			// TODO: fill transfer request
		},
	)
	if err != nil {
		return nil, err
	}

	return transferToPayment(resp)
}

func transferToPayment(transfer *client.TransferResponse) (*models.PSPPayment, error) {
	// TODO: translate transfer to formance payment object
	return nil, nil
}
