package currencycloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/currencycloud/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) validateTransferRequest(pi models.PSPPaymentInitiation) error {
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

	return nil
}

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (models.PSPPayment, error) {
	if err := p.validateTransferRequest(pi); err != nil {
		return models.PSPPayment{}, err
	}

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return models.PSPPayment{}, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %v", err),
			models.ErrInvalidRequest,
		)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return models.PSPPayment{}, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get string amount from big int amount %v: %v", pi.Amount, err),
			models.ErrInvalidRequest,
		)
	}

	resp, err := p.client.InitiateTransfer(
		ctx,
		&client.TransferRequest{
			SourceAccountID:      pi.SourceAccount.Reference,
			DestinationAccountID: pi.DestinationAccount.Reference,
			Currency:             curr,
			Amount:               json.Number(amount),
			Reason:               pi.Description,
			UniqueRequestID:      pi.Reference,
		},
	)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return translateTransferToPayment(resp)
}

func translateTransferToPayment(from *client.TransferResponse) (models.PSPPayment, error) {
	raw, err := json.Marshal(from)
	if err != nil {
		return models.PSPPayment{}, err
	}

	precision, ok := supportedCurrenciesWithDecimal[from.Currency]
	if !ok {
		return models.PSPPayment{}, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(from.Amount.String(), precision)
	if err != nil {
		return models.PSPPayment{}, err
	}

	return models.PSPPayment{
		Reference:                   from.ID,
		CreatedAt:                   from.CreatedAt,
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Amount:                      amount,
		Asset:                       currency.FormatAsset(supportedCurrenciesWithDecimal, from.Currency),
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		Status:                      matchTransactionStatus(from.Status),
		SourceAccountReference:      &from.SourceAccountID,
		DestinationAccountReference: &from.DestinationAccountID,
		Raw:                         raw,
	}, nil
}
