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

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_creditor_bank_account")

	setAsDefaultPayoutAccount := false

	currency := ba.Metadata[GocardlessCurrencyMetadataKey]
	branchCode := ba.Metadata[GocardlessBranchCodeMetadataKey]
	accountType := ba.Metadata[GocardlessAccountTypeMetadataKey]

	bankAccount, err := c.service.CreateGocardlessCreditorBankAccount(ctx, gocardless.CreditorBankAccountCreateParams{
		AccountHolderName:         ba.Name,
		AccountNumber:             *ba.AccountNumber,
		AccountType:               accountType,
		BankCode:                  *ba.SwiftBicCode,
		BranchCode:                branchCode,
		CountryCode:               *ba.Country,
		Currency:                  currency,
		Iban:                      *ba.IBAN,
		Links:                     gocardless.CreditorBankAccountCreateParamsLinks{Creditor: creditor},
		SetAsDefaultPayoutAccount: setAsDefaultPayoutAccount,
	})

	if err != nil {
		return nil, err
	}

	return bankAccount, nil

}

func (c *client) CreateCustomerBankAccount(ctx context.Context, customer string, ba models.BankAccount) (
	*gocardless.CustomerBankAccount, error,
) {

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_customer_bank_account")

	currency := ba.Metadata[GocardlessCurrencyMetadataKey]
	branchCode := ba.Metadata[GocardlessBranchCodeMetadataKey]
	accountType := ba.Metadata[GocardlessAccountTypeMetadataKey]

	bankAccount, err := c.service.CreateGocardlessCustomerBankAccount(ctx, gocardless.CustomerBankAccountCreateParams{
		AccountHolderName: ba.Name,
		AccountNumber:     *ba.AccountNumber,
		AccountType:       accountType,
		BankCode:          *ba.SwiftBicCode,
		BranchCode:        branchCode,
		CountryCode:       *ba.Country,
		Currency:          currency,
		Iban:              *ba.IBAN,
		Links:             gocardless.CustomerBankAccountCreateParamsLinks{Customer: customer},
	})

	if err != nil {
		return nil, err
	}

	return bankAccount, nil

}
