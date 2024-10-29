package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/stripe/stripe-go/v79"
)

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (models.PSPPayment, error) {
	if err := p.validatePayoutTransferRequest(pi); err != nil {
		return models.PSPPayment{}, err
	}

	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to get currency and precision from asset: %v: %w", err, models.ErrInvalidRequest)
	}

	var source *string = nil
	if pi.SourceAccount != nil && pi.SourceAccount.Reference != rootAccountReference {
		source = &pi.SourceAccount.Reference
	}

	resp, err := p.client.CreateTransfer(
		ctx,
		&client.CreateTransferRequest{
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

	payment, err := fromTransferToPayment(resp, source, &pi.DestinationAccount.Reference)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return payment, nil
}

func fromTransferToPayment(from *stripe.Transfer, source, destination *string) (models.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return models.PSPPayment{
		Reference:                   from.BalanceTransaction.ID,
		CreatedAt:                   time.Unix(from.Created, 0),
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Amount:                      big.NewInt(from.Amount),
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, strings.ToUpper(string(from.Currency))),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      models.PAYMENT_STATUS_SUCCEEDED,
		SourceAccountReference:      source,
		DestinationAccountReference: destination,
		Metadata:                    from.Metadata,
		Raw:                         raw,
	}, nil
}
