package increase

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferPayoutRequests(pi); err != nil {
		return nil, err
	}

	resp, err := p.client.InitiateTransfer(
		ctx,
		&client.TransferRequest{
			SourceAccountID:      pi.SourceAccount.ID,
			DestinationAccountID: pi.DestinationAccount.ID,
			Amount:              pi.Amount.Int64(),
			Currency:            pi.Asset,
			Description:         pi.Description,
		},
	)
	if err != nil {
		return nil, err
	}

	return transferToPayment(resp)
}

func transferToPayment(transfer *client.TransferResponse) (*models.PSPPayment, error) {
	return &models.PSPPayment{
		ID:             transfer.ID,
		CreatedAt:      transfer.CreatedAt,
		Reference:      transfer.ID,
		Amount:         transfer.Amount,
		Type:          models.PaymentTypeTransfer,
		Status:        models.PaymentStatusSucceeded,
		Scheme:        models.PaymentSchemeACH,
		Asset:         transfer.Currency,
		RawData:       transfer,
	}, nil
}
