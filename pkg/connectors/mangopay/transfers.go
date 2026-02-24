package mangopay

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/mangopay/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/google/uuid"
)

func (p *Plugin) validateTransferRequest(pi connector.PSPPaymentInitiation) error {
	_, err := uuid.Parse(pi.Reference)
	if err != nil {
		return connector.NewWrappedError(
			fmt.Errorf("reference %s is required to be an uuid in transfer request", pi.Reference),
			connector.ErrInvalidRequest,
		)
	}

	if pi.SourceAccount == nil {
		return connector.NewWrappedError(
			fmt.Errorf("source account is required in transfer request"),
			connector.ErrInvalidRequest,
		)
	}

	if pi.DestinationAccount == nil {
		return connector.NewWrappedError(
			fmt.Errorf("destination account is required in transfer request"),
			connector.ErrInvalidRequest,
		)
	}

	_, ok := pi.SourceAccount.Metadata[userIDMetadataKey]
	if !ok {
		return connector.NewWrappedError(
			fmt.Errorf("source account metadata with user id is required in transfer request"),
			connector.ErrInvalidRequest,
		)
	}

	return nil
}

func (p *Plugin) createTransfer(ctx context.Context, pi connector.PSPPaymentInitiation) (connector.PSPPayment, error) {
	if err := p.validateTransferRequest(pi); err != nil {
		return connector.PSPPayment{}, err
	}

	userID := pi.SourceAccount.Metadata[userIDMetadataKey]

	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return connector.PSPPayment{}, connector.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %v", err),
			connector.ErrInvalidRequest,
		)
	}

	resp, err := p.client.InitiateWalletTransfer(
		ctx,
		&client.TransferRequest{
			Reference: pi.Reference,
			AuthorID:  userID,
			DebitedFunds: client.Funds{
				Currency: curr,
				Amount:   json.Number(pi.Amount.String()),
			},
			Fees: client.Funds{
				Currency: curr,
				Amount:   "0",
			},
			DebitedWalletID:  pi.SourceAccount.Reference,
			CreditedWalletID: pi.DestinationAccount.Reference,
		},
	)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	payment, err := FromTransferToPayment(resp)
	if err != nil {
		return connector.PSPPayment{}, err
	}

	return payment, nil
}

func FromTransferToPayment(from *client.TransferResponse) (connector.PSPPayment, error) {
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
		Type:                        connector.PAYMENT_TYPE_TRANSFER,
		Amount:                      &amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, from.DebitedFunds.Currency),
		Scheme:                      connector.PAYMENT_SCHEME_OTHER,
		Status:                      matchPaymentStatus(from.Status),
		SourceAccountReference:      &from.DebitedWalletID,
		DestinationAccountReference: &from.CreditedWalletID,
		Raw:                         raw,
	}, nil
}
