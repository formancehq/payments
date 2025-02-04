package increase

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

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
			AccountID:   pi.SourceAccount.Reference,
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
		Reference: from.ID,
		CreatedAt: time.Now(),
		Amount:    big.NewInt(from.Amount),
		Type:      models.PAYMENT_TYPE_PAYOUT,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Scheme:    models.PAYMENT_SCHEME_ACH,
		Asset:     from.Currency,
		Raw:       json.RawMessage([]byte(`{"id":"` + from.ID + `"}`)),
	}, nil
}
