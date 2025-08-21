package checkout

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/plugins/public/checkout/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	// TODO: Since we are in minor units currency in the PSPPaymentInitiation
	// object, we sometimes need to put back the amount in float for
	// the PSP. You can use the next methods to do that.
	// curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	// if err != nil {
	//	return nil, fmt.Errorf("failed to get currency and precision from asset: %v: %w", err, models.ErrInvalidRequest)
	//}

	// amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	// if err != nil {
	//	 return nil, fmt.Errorf("failed to get string amount from big int: %v: %w", err, models.ErrInvalidRequest)
	// }

	resp, err := p.client.InitiatePayout(
		ctx,
		&client.PayoutRequest{
			// TODO: fill payout request
		},
	)
	if err != nil {
		return nil, err
	}

	return payoutToPayment(resp)
}

func payoutToPayment(from *client.PayoutResponse) (*models.PSPPayment, error) {
	// TODO: translate payout to formance payment object
	return nil, nil
}
