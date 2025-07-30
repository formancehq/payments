package bankingcircle

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingcircle/client"
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

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferRequest(pi); err != nil {
		return nil, err
	}

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from asset: %w", err),
			models.ErrInvalidRequest,
		)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get string amount from big int amount %v: %v", pi.Amount, err),
			models.ErrInvalidRequest,
		)
	}

	var sourceAccount *client.Account
	sourceAccount, err = p.client.GetAccount(ctx, pi.SourceAccount.Reference)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get source account %s: %v", pi.SourceAccount.Reference, err),
			models.ErrInvalidRequest,
		)
	}
	if len(sourceAccount.AccountIdentifiers) == 0 {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("no account identifiers provided for source account %s", pi.SourceAccount.Reference),
			models.ErrInvalidRequest,
		)
	}

	var destinationAccount *client.Account
	destinationAccount, err = p.client.GetAccount(ctx, pi.DestinationAccount.Reference)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get destination account %s: %v", pi.DestinationAccount.Reference, err),
			models.ErrInvalidRequest,
		)
	}
	if len(destinationAccount.AccountIdentifiers) == 0 {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("no account identifiers provided for destination account %s", pi.DestinationAccount.Reference),
			models.ErrInvalidRequest,
		)
	}

	resp, err := p.client.InitiateTransferOrPayouts(
		ctx,
		&client.PaymentRequest{
			IdempotencyKey:         pi.Reference,
			RequestedExecutionDate: pi.CreatedAt,
			DebtorAccount: client.PaymentAccount{
				Account:              sourceAccount.AccountIdentifiers[0].Account,
				FinancialInstitution: sourceAccount.AccountIdentifiers[0].FinancialInstitution,
				Country:              sourceAccount.AccountIdentifiers[0].Country,
			},
			DebtorReference:    pi.Description,
			CurrencyOfTransfer: curr,
			Amount: struct {
				Currency string      "json:\"currency\""
				Amount   json.Number "json:\"amount\""
			}{
				Currency: curr,
				Amount:   json.Number(amount),
			},
			ChargeBearer: "SHA",
			CreditorAccount: &client.PaymentAccount{
				Account:              destinationAccount.AccountIdentifiers[0].Account,
				FinancialInstitution: destinationAccount.AccountIdentifiers[0].FinancialInstitution,
				Country:              destinationAccount.AccountIdentifiers[0].Country,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	payment, err := p.client.GetPayment(ctx, resp.PaymentID)
	if err != nil {
		return nil, err
	}

	res, err := translatePayment(*payment)
	if err != nil {
		return nil, err
	}

	return res, nil
}
