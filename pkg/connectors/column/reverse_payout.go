package column

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/pkg/connectors/column/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) createReversePayout(ctx context.Context, pr connector.PSPPaymentInitiationReversal) (connector.ReversePayoutResponse, error) {
	if err := p.validateReversePayout(pr); err != nil {
		return connector.ReversePayoutResponse{}, err
	}

	resp, err := p.client.ReversePayout(ctx, &client.ReversePayoutRequest{
		ACHTransferID: pr.RelatedPaymentInitiation.Reference,
		Reason:        client.ReversePayoutReason(pr.Metadata[client.ColumnReasonMetadataKey]),
		Description:   pr.Description,
	})
	if err != nil {
		return connector.ReversePayoutResponse{}, err
	}

	payment, err := p.transformReversePayoutResponse(resp)
	if err != nil {
		return connector.ReversePayoutResponse{}, err
	}

	return connector.ReversePayoutResponse{
		Payment: payment,
	}, nil
}

func (p *Plugin) transformReversePayoutResponse(resp *client.ReversePayoutResponse) (connector.PSPPayment, error) {
	raw, err := json.Marshal(resp)

	if err != nil {
		return connector.PSPPayment{}, err
	}

	createdAt, err := ParseColumnTimestamp(resp.CreatedAt)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	return connector.PSPPayment{
		Amount:                      big.NewInt(resp.Amount),
		Asset:                       resp.CurrencyCode,
		Status:                      p.mapTransactionStatus(resp.Status),
		Raw:                         raw,
		Reference:                   resp.ID,
		CreatedAt:                   createdAt,
		Type:                        connector.PAYMENT_TYPE_TRANSFER,
		SourceAccountReference:      pointer.For(resp.BankAccountID),
		DestinationAccountReference: pointer.For(resp.CounterpartyID),
	}, nil
}
