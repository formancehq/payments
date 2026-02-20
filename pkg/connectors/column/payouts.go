package column

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/column/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) createPayout(ctx context.Context, pi connector.PSPPaymentInitiation) (connector.CreatePayoutResponse, error) {
	if err := p.validatePayoutRequests(pi); err != nil {
		return connector.CreatePayoutResponse{}, err
	}
	var curr string
	if pi.Asset != "" {

		currency, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
		if err != nil {
			return connector.CreatePayoutResponse{}, fmt.Errorf("failed to get currency and precision from asset: %v: %w", err, connector.ErrInvalidRequest)
		}
		curr = currency
	}

	resp, err := p.client.InitiatePayout(
		ctx,
		&client.PayoutRequest{
			Amount:             pi.Amount.Int64(),
			CurrencyCode:       curr,
			Metadata:           pi.Metadata,
			SourceAccount:      pi.SourceAccount.Reference,
			DestinationAccount: pi.DestinationAccount.Reference,
			Description:        pi.Description,
		},
	)
	if err != nil {
		return connector.CreatePayoutResponse{}, err
	}

	payment, err := p.payoutToPayment(resp.ID, resp)
	if err != nil {
		return connector.CreatePayoutResponse{}, err
	}

	return connector.CreatePayoutResponse{
		Payment: payment,
	}, nil
}

func (p *Plugin) payoutToPayment(id string, from *client.PayoutResponse) (*connector.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return &connector.PSPPayment{}, err
	}

	createdAt := time.Time{}
	if from.CreatedAt != "" {
		createdAt, err = ParseColumnTimestamp(from.CreatedAt)
		if err != nil {
			return &connector.PSPPayment{}, err
		}
	}

	curr := ""
	if from.CurrencyCode != "" {
		curr = currency.FormatAsset(supportedCurrenciesWithDecimal, from.CurrencyCode)
	}

	return &connector.PSPPayment{
		Reference:                   id,
		ParentReference:             from.ID,
		Amount:                      big.NewInt(from.Amount),
		Asset:                       curr,
		Status:                      p.mapTransactionStatus(from.Status),
		Raw:                         raw,
		Type:                        mapPayoutType(from),
		SourceAccountReference:      pointer.For(from.BankAccountID),
		DestinationAccountReference: pointer.For(from.CounterpartyId),
		CreatedAt:                   createdAt,
		Metadata:                    from.Metadata,
	}, nil
}

func mapPayoutType(payout *client.PayoutResponse) connector.PaymentType {
	if payout.IsIncoming {
		return connector.PAYMENT_TYPE_PAYIN
	} else {
		return connector.PAYMENT_TYPE_PAYOUT
	}
}
