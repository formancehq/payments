package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/stripe/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/stripe/stripe-go/v80"
)

func (p *Plugin) createPayout(ctx context.Context, pi connector.PSPPaymentInitiation) (connector.PSPPayment, error) {
	if err := p.validatePayoutTransferRequest(pi); err != nil {
		return connector.PSPPayment{}, err
	}

	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return connector.PSPPayment{}, connector.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %w", err),
			connector.ErrInvalidRequest,
		)
	}

	var source *string = nil
	if pi.SourceAccount != nil && pi.SourceAccount.Reference != p.client.GetRootAccountID() {
		source = &pi.SourceAccount.Reference
	}

	resp, err := p.client.CreatePayout(
		ctx,
		&client.CreatePayoutRequest{
			IdempotencyKey: pi.Reference,
			Amount:         pi.Amount.Int64(),
			Currency:       curr,
			Source:         source,
			Destination:    pi.DestinationAccount.Reference,
			Description:    pi.Description,
			Metadata:       pi.Metadata,
		},
	)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	payment, err := fromPayoutToPayment(resp, source, &pi.DestinationAccount.Reference)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	return payment, nil
}

func fromPayoutToPayment(from *stripe.Payout, source, destination *string) (connector.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	return connector.PSPPayment{
		Reference:                   from.BalanceTransaction.ID,
		CreatedAt:                   time.Unix(from.Created, 0),
		Type:                        connector.PAYMENT_TYPE_PAYOUT,
		Amount:                      big.NewInt(from.Amount),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, strings.ToUpper(string(from.Currency))),
		Scheme:                      connector.PAYMENT_SCHEME_OTHER,
		Status:                      matchPayoutStatus(from.Status),
		SourceAccountReference:      source,
		DestinationAccountReference: destination,
		Metadata:                    from.Metadata,
		Raw:                         raw,
	}, nil
}

func matchPayoutStatus(status stripe.PayoutStatus) connector.PaymentStatus {
	switch status {
	case stripe.PayoutStatusCanceled:
		return connector.PAYMENT_STATUS_CANCELLED
	case stripe.PayoutStatusFailed:
		return connector.PAYMENT_STATUS_FAILED
	case stripe.PayoutStatusInTransit, stripe.PayoutStatusPending:
		return connector.PAYMENT_STATUS_PENDING
	case stripe.PayoutStatusPaid:
		return connector.PAYMENT_STATUS_SUCCEEDED
	default:
		return connector.PAYMENT_STATUS_UNKNOWN
	}
}
