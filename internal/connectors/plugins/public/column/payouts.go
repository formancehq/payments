package column

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (models.CreatePayoutResponse, error) {
	if err := p.validatePayoutRequests(pi); err != nil {
		return models.CreatePayoutResponse{}, err
	}
	var curr string
	if pi.Asset != "" {

		currency, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
		if err != nil {
			return models.CreatePayoutResponse{}, fmt.Errorf("failed to get currency and precision from asset: %v: %w", err, models.ErrInvalidRequest)
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
		return models.CreatePayoutResponse{}, err
	}

	payment, err := p.payoutToPayment(resp)
	if err != nil {
		return models.CreatePayoutResponse{}, err
	}

	return models.CreatePayoutResponse{
		Payment: payment,
	}, nil
}

func (p *Plugin) payoutToPayment(from *client.PayoutResponse) (*models.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return &models.PSPPayment{}, err
	}

	createdAt := time.Time{}
	if from.CreatedAt != "" {
		createdAt, err = ParseColumnTimestamp(from.CreatedAt)
		if err != nil {
			return &models.PSPPayment{}, err
		}
	}

	curr := ""
	if from.CurrencyCode != "" {
		curr = currency.FormatAsset(supportedCurrenciesWithDecimal, from.CurrencyCode)
	}

	return &models.PSPPayment{
		Amount:                      big.NewInt(from.Amount),
		Asset:                       curr,
		Status:                      p.mapTransactionStatus(from.Status),
		Raw:                         raw,
		Reference:                   from.ID,
		Type:                        mapPayoutType(from),
		SourceAccountReference:      pointer.For(from.BankAccountID),
		DestinationAccountReference: pointer.For(from.CounterpartyId),
		CreatedAt:                   createdAt,
		Metadata:                    from.Metadata,
	}, nil
}

func mapPayoutType(payout *client.PayoutResponse) models.PaymentType {
	if payout.IsIncoming {
		return models.PAYMENT_TYPE_PAYIN
	} else {
		return models.PAYMENT_TYPE_PAYOUT
	}
}
