package mangopay

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

var (
	bankWireRefPatternRegexp = regexp.MustCompile("[a-zA-Z0-9 ]*")
)

func (p *Plugin) validatePayoutRequest(pi models.PSPPaymentInitiation) error {
	_, err := uuid.Parse(pi.Reference)
	if err != nil {
		return fmt.Errorf("reference is required as an uuid: %w", models.ErrInvalidRequest)
	}

	if pi.SourceAccount == nil {
		return models.ErrInvalidRequest
	}

	if pi.DestinationAccount == nil {
		return models.ErrInvalidRequest
	}

	_, ok := pi.SourceAccount.Metadata[userIDMetadataKey]
	if !ok {
		return fmt.Errorf("source account metadata with user id is required: %w", models.ErrInvalidRequest)
	}

	if len(pi.Description) > 12 || !bankWireRefPatternRegexp.MatchString(pi.Description) {
		return fmt.Errorf("description must be alphanumeric and less than 12 characters: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (models.PSPPayment, error) {
	if err := p.validatePayoutRequest(pi); err != nil {
		return models.PSPPayment{}, err
	}

	userID := pi.SourceAccount.Metadata[userIDMetadataKey]

	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to get currency and precision from asset: %v: %w", err, models.ErrInvalidRequest)
	}

	resp, err := p.client.InitiatePayout(ctx, &client.PayoutRequest{
		AuthorID: userID,
		DebitedFunds: client.Funds{
			Currency: curr,
			Amount:   json.Number(pi.Amount.String()),
		},
		Fees: client.Funds{
			Currency: curr,
			Amount:   json.Number("0"),
		},
		DebitedWalletID: pi.SourceAccount.Reference,
		BankAccountID:   pi.DestinationAccount.Reference,
		BankWireRef:     pi.Description,
	})
	if err != nil {
		return models.PSPPayment{}, err
	}

	payment, err := FromPayoutToPayment(resp, pi.DestinationAccount.Reference)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return payment, nil
}

func FromPayoutToPayment(from *client.PayoutResponse, destinationAccountReference string) (models.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return models.PSPPayment{}, err
	}

	var amount big.Int
	_, ok := amount.SetString(from.DebitedFunds.Amount.String(), 10)
	if !ok {
		return models.PSPPayment{}, fmt.Errorf("failed to parse amount %s", from.DebitedFunds.Amount.String())
	}

	return models.PSPPayment{
		Reference:                   from.ID,
		CreatedAt:                   time.Unix(from.CreationDate, 0),
		Type:                        models.PAYMENT_TYPE_PAYOUT,
		Amount:                      &amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, from.DebitedFunds.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      matchPaymentStatus(from.Status),
		SourceAccountReference:      &from.DebitedWalletID,
		DestinationAccountReference: &destinationAccountReference,
		Raw:                         raw,
	}, nil
}
