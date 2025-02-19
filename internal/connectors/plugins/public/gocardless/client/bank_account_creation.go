package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"
)

func (c *client) CreateCreditorBankAccount(ctx context.Context, creditor string, ba models.BankAccount) (
	*gocardless.CreditorBankAccount, error,
) {

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_customer_bank_account")

	bankAccount, err := c.service.CreditorBankAccounts.Create(ctx, gocardless.CreditorBankAccountCreateParams{
		AccountHolderName:         ba.Name,
		AccountNumber:             *ba.AccountNumber,
		AccountType:               ba.Metadata["account_type"],
		BankCode:                  *ba.SwiftBicCode,
		BranchCode:                ba.Metadata["branch_code"],
		CountryCode:               *ba.Country,
		Currency:                  ba.Metadata["currency"],
		Iban:                      *ba.IBAN,
		Links:                     gocardless.CreditorBankAccountCreateParamsLinks{Creditor: creditor},
		SetAsDefaultPayoutAccount: ba.Metadata["set_as_default_payout_account"] == "true",
	})

	if err != nil {
		return nil, err
	}

	return bankAccount, nil

}

func (c *client) CreateCustomerBankAccount(ctx context.Context, customer string, ba models.BankAccount) (
	*gocardless.CustomerBankAccount, error,
) {

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_creditor_bank_account")

	bankAccount, err := c.service.CustomerBankAccounts.Create(ctx, gocardless.CustomerBankAccountCreateParams{
		AccountHolderName: ba.Name,
		AccountNumber:     *ba.AccountNumber,
		AccountType:       ba.Metadata["account_type"],
		BankCode:          *ba.SwiftBicCode,
		BranchCode:        ba.Metadata["branch_code"],
		CountryCode:       *ba.Country,
		Currency:          ba.Metadata["currency"],
		Iban:              *ba.IBAN,
		Links:             gocardless.CustomerBankAccountCreateParamsLinks{Customer: customer},
	})

	if err != nil {
		return nil, err
	}

	return bankAccount, nil

}
