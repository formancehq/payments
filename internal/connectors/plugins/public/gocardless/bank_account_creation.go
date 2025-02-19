package gocardless

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"
)

func validateExternalBankAccount(newExternalBankAccount models.BankAccount) error {
	_, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, client.GocardlessBranchCodeMetadataKey)

	if err != nil {
		return fmt.Errorf("required metadata field branch_code is missing")
	}

	reqCurrency, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, client.GocardlessCurrencyMetadataKey)
	if err != nil {
		return fmt.Errorf("required metadata field currency is missing")
	}

	_, ok := supportedCurrenciesWithDecimal[reqCurrency]

	if !ok {

		return fmt.Errorf("currency %s not supported", reqCurrency)
	}

	creditor, _ := extractNamespacedMetadata(newExternalBankAccount.Metadata, client.GocardlessCreditorMetadataKey)
	customer, _ := extractNamespacedMetadata(newExternalBankAccount.Metadata, client.GocardlessCustomerMetadataKey)

	if creditor != "" && creditor[:2] != "CR" {
		return fmt.Errorf("creditor ID must start with 'CR'")
	}

	if customer != "" && customer[:2] != "CU" {
		return fmt.Errorf("customer ID must start with 'CU'")
	}

	if customer == "" && creditor == "" {
		return fmt.Errorf("you must provide customer or creditor metadata field")
	}

	if customer != "" && creditor != "" {
		return fmt.Errorf("you must provide either customer or creditor metadata field but not both")
	}

	if newExternalBankAccount.Country != nil && *newExternalBankAccount.Country == "US" {

		accountType, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, client.GocardlessAccountTypeMetadataKey)

		if err != nil {
			return fmt.Errorf("required metadata field currency is missing")
		}

		if accountType != "checking" && accountType != "savings" {
			return fmt.Errorf("metadata field account_type must be checking or savings")

		}

	}

	return nil
}

func (p *Plugin) createBankAccount(ctx context.Context, ba models.BankAccount) (models.CreateBankAccountResponse, error) {
	err := validateExternalBankAccount(ba)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	creditor := ba.Metadata["creditor"]
	customer := ba.Metadata["customer"]
	var externalBankAccount client.GocardlessGenericAccount

	if creditor != "" {
		creditorBankAccount, err := p.client.CreateCreditorBankAccount(ctx, creditor, ba)

		if err != nil {
			return models.CreateBankAccountResponse{}, err
		}

		parsedTime, err := ParseGocardlessTimestamp(creditorBankAccount.CreatedAt)
		if err != nil {
			return models.CreateBankAccountResponse{}, fmt.Errorf("failed to parse creation time: %w", err)
		}

		externalBankAccount = client.GocardlessGenericAccount{
			ID:                creditorBankAccount.Id,
			CreatedAt:         parsedTime.Unix(),
			AccountHolderName: creditorBankAccount.AccountHolderName,
			Metadata:          creditorBankAccount.Metadata,
			Currency:          creditorBankAccount.Currency,
			AccountType:       creditorBankAccount.AccountType,
		}
	}

	if customer != "" {
		customerBankAccount, err := p.client.CreateCustomerBankAccount(ctx, customer, ba)

		if err != nil {
			return models.CreateBankAccountResponse{}, err
		}

		parsedTime, err := ParseGocardlessTimestamp(customerBankAccount.CreatedAt)
		if err != nil {
			return models.CreateBankAccountResponse{}, fmt.Errorf("failed to parse creation time: %w", err)
		}

		externalBankAccount = client.GocardlessGenericAccount{
			ID:                customerBankAccount.Id,
			CreatedAt:         parsedTime.Unix(),
			AccountHolderName: customerBankAccount.AccountHolderName,
			Metadata:          customerBankAccount.Metadata,
			Currency:          customerBankAccount.Currency,
			AccountType:       customerBankAccount.AccountType,
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
