package column

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createReversePayout(ctx context.Context, pr models.PSPPaymentInitiationReversal) (models.ReversePayoutResponse, error) {
	if err := p.validateReversePayout(pr); err != nil {
		return models.ReversePayoutResponse{}, err
	}

	resp, err := p.client.ReversePayout(ctx, &client.ReversePayoutRequest{
		ACHTransferID: pr.RelatedPaymentInitiation.Reference,
		Reason:        client.ReversePayoutReason(pr.Metadata[client.ColumnReasonMetadataKey]),
		Description:   pr.Description,
	})
	if err != nil {
		return models.ReversePayoutResponse{}, err
	}

	payment, err := p.transformReversePayoutResponse(resp)
	if err != nil {
		return models.ReversePayoutResponse{}, err
	}

	return models.ReversePayoutResponse{
		Payment: payment,
	}, nil
}

func (p *Plugin) transformReversePayoutResponse(resp *client.ReversePayoutResponse) (models.PSPPayment, error) {
	raw, err := json.Marshal(resp)

	if err != nil {
		return models.PSPPayment{}, err
	}

	createdAt, err := ParseColumnTimestamp(resp.CreatedAt)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return models.PSPPayment{
		Amount:                      big.NewInt(resp.Amount),
		Asset:                       resp.CurrencyCode,
		Status:                      p.mapTransactionStatus(resp.Status),
		Raw:                         raw,
		Reference:                   resp.ID,
		CreatedAt:                   createdAt,
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		SourceAccountReference:      pointer.For(resp.BankAccountID),
		DestinationAccountReference: pointer.For(resp.CounterpartyID),
	}, nil
}
