package increase

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

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
			SourceAccountID:      pi.SourceAccount.Reference,
			DestinationAccountID: pi.DestinationAccount.Reference,
			Amount:               pi.Amount.Int64(),
			Currency:             pi.Asset,
			Description:          pi.Description,
		},
	)
	if err != nil {
		return nil, err
	}

	return transferToPayment(resp)
}

func transferToPayment(transfer *client.TransferResponse) (*models.PSPPayment, error) {
	return &models.PSPPayment{
		Reference: transfer.ID,
		CreatedAt: time.Now(),
		Amount:    big.NewInt(transfer.Amount),
		Type:      models.PAYMENT_TYPE_TRANSFER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Scheme:    models.PAYMENT_SCHEME_ACH,
		Asset:     transfer.Currency,
		Raw:       json.RawMessage([]byte(`{"id":"` + transfer.ID + `"}`)),
	}, nil
}
