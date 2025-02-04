package increase

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	resp, err := p.client.InitiatePayout(
		ctx,
		&client.PayoutRequest{
			AccountID:    pi.SourceAccount.ID,
			Amount:      pi.Amount.Int64(),
			Currency:    pi.Asset,
			Description: pi.Description,
		},
	)
	if err != nil {
		return nil, err
	}

	return payoutToPayment(resp)
}

func payoutToPayment(from *client.PayoutResponse) (*models.PSPPayment, error) {
	return &models.PSPPayment{
		ID:             from.ID,
		CreatedAt:      from.CreatedAt,
		Reference:      from.ID,
		Amount:         from.Amount,
		Type:          models.PaymentTypePayout,
		Status:        models.PaymentStatusSucceeded,
		Scheme:        models.PaymentSchemeACH,
		Asset:         from.Currency,
		RawData:       from,
	}, nil
}
