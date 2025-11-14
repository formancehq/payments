package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/stripe/stripe-go/v80"
)

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (models.PSPPayment, error) {
	if err := p.validatePayoutTransferRequest(pi); err != nil {
		return models.PSPPayment{}, err
	}

	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return models.PSPPayment{}, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %w", err),
			models.ErrInvalidRequest,
		)
	}

	var source *string = nil
	if pi.SourceAccount != nil && pi.SourceAccount.Reference != rootAccountReference {
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
		return models.PSPPayment{}, err
	}

	payment, err := fromPayoutToPayment(resp, source, &pi.DestinationAccount.Reference)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return payment, nil
}

func fromPayoutToPayment(from *stripe.Payout, source, destination *string) (models.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return models.PSPPayment{
		Reference:                   from.BalanceTransaction.ID,
		CreatedAt:                   time.Unix(from.Created, 0),
		Type:                        models.PAYMENT_TYPE_PAYOUT,
		Amount:                      big.NewInt(from.Amount),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, strings.ToUpper(string(from.Currency))),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      matchPayoutStatus(from.Status),
		SourceAccountReference:      source,
		DestinationAccountReference: destination,
		Metadata:                    from.Metadata,
		Raw:                         raw,
	}, nil
}

func matchPayoutStatus(status stripe.PayoutStatus) models.PaymentStatus {
	switch status {
	case stripe.PayoutStatusCanceled:
		return models.PAYMENT_STATUS_CANCELLED
	case stripe.PayoutStatusFailed:
		return models.PAYMENT_STATUS_FAILED
	case stripe.PayoutStatusInTransit, stripe.PayoutStatusPending:
		return models.PAYMENT_STATUS_PENDING
	case stripe.PayoutStatusPaid:
		return models.PAYMENT_STATUS_SUCCEEDED
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}
