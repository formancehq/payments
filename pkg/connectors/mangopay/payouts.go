package mangopay

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/mangopay/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/google/uuid"
)

var (
	bankWireRefPatternRegexp = regexp.MustCompile("[a-zA-Z0-9 ]*")
)

func (p *Plugin) validatePayoutRequest(pi connector.PSPPaymentInitiation) error {
	_, err := uuid.Parse(pi.Reference)
	if err != nil {
		return connector.NewWrappedError(
			fmt.Errorf("reference %s is required to be an uuid in payout request", pi.Reference),
			connector.ErrInvalidRequest,
		)
	}

	if pi.SourceAccount == nil {
		return connector.NewWrappedError(
			fmt.Errorf("source account is required in payout request"),
			connector.ErrInvalidRequest,
		)
	}

	if pi.DestinationAccount == nil {
		return connector.NewWrappedError(
			fmt.Errorf("destination account is required in payout request"),
			connector.ErrInvalidRequest,
		)
	}

	_, ok := pi.SourceAccount.Metadata[userIDMetadataKey]
	if !ok {
		return connector.NewWrappedError(
			fmt.Errorf("source account metadata with user id is required in payout request"),
			connector.ErrInvalidRequest,
		)
	}

	if len(pi.Description) > 12 || !bankWireRefPatternRegexp.MatchString(pi.Description) {
		return connector.NewWrappedError(
			fmt.Errorf("description must be alphanumeric and less than 12 characters in payout request"),
			connector.ErrInvalidRequest,
		)
	}

	return nil
}

func (p *Plugin) createPayout(ctx context.Context, pi connector.PSPPaymentInitiation) (connector.PSPPayment, error) {
	if err := p.validatePayoutRequest(pi); err != nil {
		return connector.PSPPayment{}, err
	}

	userID := pi.SourceAccount.Metadata[userIDMetadataKey]

	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return connector.PSPPayment{}, connector.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %w", err),
			connector.ErrInvalidRequest,
		)
	}

	resp, err := p.client.InitiatePayout(ctx, &client.PayoutRequest{
		Reference: pi.Reference,
		AuthorID:  userID,
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
		return connector.PSPPayment{}, err
	}

	payment, err := FromPayoutToPayment(resp, pi.DestinationAccount.Reference)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	return payment, nil
}

func FromPayoutToPayment(from *client.PayoutResponse, destinationAccountReference string) (connector.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	var amount big.Int
	_, ok := amount.SetString(from.DebitedFunds.Amount.String(), 10)
	if !ok {
		return connector.PSPPayment{}, fmt.Errorf("failed to parse amount %s", from.DebitedFunds.Amount.String())
	}

	return connector.PSPPayment{
		Reference:                   from.ID,
		CreatedAt:                   time.Unix(from.CreationDate, 0),
		Type:                        connector.PAYMENT_TYPE_PAYOUT,
		Amount:                      &amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, from.DebitedFunds.Currency),
		Scheme:                      connector.PAYMENT_SCHEME_OTHER,
		Status:                      matchPaymentStatus(from.Status),
		SourceAccountReference:      &from.DebitedWalletID,
		DestinationAccountReference: &destinationAccountReference,
		Raw:                         raw,
	}, nil
}
