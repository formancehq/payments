package bankingcircle

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingcircle/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) validateTransferRequest(pi models.PSPPaymentInitiation) error {
	if pi.SourceAccount == nil {
		return fmt.Errorf("source account is required: %w", models.ErrInvalidRequest)
	}

	if pi.DestinationAccount == nil {
		return fmt.Errorf("destination account is required: %w", models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) createTransfer(ctx context.Context, pi models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	if err := p.validateTransferRequest(pi); err != nil {
		return nil, err
	}

	curr, precision, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, pi.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed to get currency and precision from asset: %v: %w", err, models.ErrInvalidRequest)
	}

	amount, err := currency.GetStringAmountFromBigIntWithPrecision(pi.Amount, precision)
	if err != nil {
		return nil, fmt.Errorf("failed to get string amount from big int: %v: %w", err, models.ErrInvalidRequest)
	}

	var sourceAccount *client.Account
	sourceAccount, err = p.client.GetAccount(ctx, pi.SourceAccount.Reference)
	if err != nil {
		return nil, fmt.Errorf("failed to get source account: %v: %w", err, models.ErrInvalidRequest)
	}
	if len(sourceAccount.AccountIdentifiers) == 0 {
		return nil, fmt.Errorf("no account identifiers provided for source account: %w", models.ErrInvalidRequest)
	}

	var destinationAccount *client.Account
	destinationAccount, err = p.client.GetAccount(ctx, pi.DestinationAccount.Reference)
	if err != nil {
		return nil, fmt.Errorf("failed to get destination account: %v: %w", err, models.ErrInvalidRequest)
	}
	if len(destinationAccount.AccountIdentifiers) == 0 {
		return nil, fmt.Errorf("no account identifiers provided for destination account: %w", models.ErrInvalidRequest)
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
