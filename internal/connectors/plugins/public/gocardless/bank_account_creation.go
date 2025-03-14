package gocardless

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createBankAccount(ctx context.Context, ba models.BankAccount) (models.CreateBankAccountResponse, error) {
	err := validateExternalBankAccount(ba)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	creditor := ba.Metadata[client.GocardlessCreditorMetadataKey]
	customer := ba.Metadata[client.GocardlessCustomerMetadataKey]
	var externalBankAccount client.GocardlessGenericAccount

	if creditor != "" {
		externalBankAccount, err = p.client.CreateCreditorBankAccount(ctx, creditor, ba)

		if err != nil {
			return models.CreateBankAccountResponse{}, err
		}
	}

	if customer != "" {
		externalBankAccount, err = p.client.CreateCustomerBankAccount(ctx, customer, ba)

		if err != nil {
			return models.CreateBankAccountResponse{}, err
		}
	}

	bankAccount, err := externalAccountFromGocardlessData(externalBankAccount)

	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	return models.CreateBankAccountResponse{
		RelatedAccount: bankAccount,
	}, nil
}
