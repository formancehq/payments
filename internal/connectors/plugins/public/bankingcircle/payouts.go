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

func (p *Plugin) validatePayoutRequest(pi models.PSPPaymentInitiation) error {
	if pi.SourceAccount == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("source account is required in payout request"),
			models.ErrInvalidRequest,
		)
	}

	if pi.DestinationAccount == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("destination account is required in payout request"),
			models.ErrInvalidRequest,
		)
	}

	if pi.DestinationAccount.Name == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("destination account name is required in payout request"),
			models.ErrInvalidRequest,
		)
	}

	if pi.DestinationAccount.Metadata[models.AccountAccountNumberMetadataKey] == "" &&
		pi.DestinationAccount.Metadata[models.AccountIBANMetadataKey] == "" {
		return errorsutils.NewWrappedError(
			fmt.Errorf("destination account number or IBAN is required in payout request"),
			models.ErrInvalidRequest,
		)
	}

	return nil
}

func (p *Plugin) createPayout(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validatePayoutRequest(pi); err != nil {
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

	account := pi.DestinationAccount.Metadata[models.AccountAccountNumberMetadataKey]
	if account == "" {
		account = pi.DestinationAccount.Metadata[models.AccountIBANMetadataKey]
	}

	resp, err := p.client.InitiateTransferOrPayouts(ctx, &client.PaymentRequest{
		IdempotencyKey:         pi.Reference,
		RequestedExecutionDate: pi.CreatedAt,
		DebtorAccount: client.PaymentAccount{
			Account:              sourceAccount.AccountIdentifiers[0].Account,
			FinancialInstitution: sourceAccount.AccountIdentifiers[0].FinancialInstitution,
			Country:              sourceAccount.AccountIdentifiers[0].Country,
		},
		DebtorReference:    pi.Description,
		CurrencyOfTransfer: curr,
		Amount: client.Amount{
			Currency: curr,
			Amount:   json.Number(amount),
		},
		ChargeBearer: "SHA",
		CreditorAccount: &client.PaymentAccount{
			Account:              account,
			FinancialInstitution: pi.DestinationAccount.Metadata[models.AccountSwiftBicCodeMetadataKey],
			Country:              pi.DestinationAccount.Metadata[models.AccountBankAccountCountryMetadataKey],
		},
		CreditorName: *pi.DestinationAccount.Name,
	})
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
