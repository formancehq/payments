package mangopay

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/google/uuid"
)

func (p *Plugin) validateTransferRequest(pi models.PSPPaymentInitiation) error {
	_, err := uuid.Parse(pi.Reference)
	if err != nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("reference %s is required to be an uuid in transfer request", pi.Reference),
			models.ErrInvalidRequest,
		)
	}

	if pi.SourceAccount == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("source account is required in transfer request"),
			models.ErrInvalidRequest,
		)
	}

	if pi.DestinationAccount == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("destination account is required in transfer request"),
			models.ErrInvalidRequest,
		)
	}

	_, ok := pi.SourceAccount.Metadata[userIDMetadataKey]
	if !ok {
		return errorsutils.NewWrappedError(
			fmt.Errorf("source account metadata with user id is required in transfer request"),
			models.ErrInvalidRequest,
		)
	}

	return nil
}

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (models.PSPPayment, error) {
	if err := p.validateTransferRequest(pi); err != nil {
		return models.PSPPayment{}, err
	}

	userID := pi.SourceAccount.Metadata[userIDMetadataKey]

	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return models.PSPPayment{}, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %v", err),
			models.ErrInvalidRequest,
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
		return models.PSPPayment{}, err
	}

	payment, err := FromTransferToPayment(resp)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return payment, nil
}

func FromTransferToPayment(from *client.TransferResponse) (models.PSPPayment, error) {
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
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Amount:                      &amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, from.DebitedFunds.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      matchPaymentStatus(from.Status),
		SourceAccountReference:      &from.DebitedWalletID,
		DestinationAccountReference: &from.CreditedWalletID,
		Raw:                         raw,
	}, nil
}
