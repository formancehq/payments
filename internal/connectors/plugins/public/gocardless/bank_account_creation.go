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
		return fmt.Errorf("required metadata field com.gocardless.spec/branch_code is missing")
	}

	reqCurrency, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, client.GocardlessCurrencyMetadataKey)
	if err != nil {
		return fmt.Errorf("required metadata field com.gocardless.spec/currency is missing")
	}

	_, ok := SupportedCurrenciesWithDecimal[reqCurrency]

	if !ok {

		return fmt.Errorf("com.gocardless.spec/currency %s not supported", reqCurrency)
	}

	creditor, _ := extractNamespacedMetadata(newExternalBankAccount.Metadata, client.GocardlessCreditorMetadataKey)
	customer, _ := extractNamespacedMetadata(newExternalBankAccount.Metadata, client.GocardlessCustomerMetadataKey)

	if len(creditor) > 1 && creditor[:2] != "CR" {
		return fmt.Errorf("com.gocardless.spec/creditor ID must start with 'CR'")
	}

	if len(customer) > 1 && customer[:2] != "CU" {
		return fmt.Errorf("com.gocardless.spec/customer ID must start with 'CU'")
	}

	if customer == "" && creditor == "" {
		return fmt.Errorf("you must provide com.gocardless.spec/customer or com.gocardless.spec/creditor metadata field")
	}

	if customer != "" && creditor != "" {
		return fmt.Errorf("you must provide either com.gocardless.spec/customer or com.gocardless.spec/creditor metadata field but not both")
	}

	if newExternalBankAccount.Country != nil && *newExternalBankAccount.Country == "US" {

		accountType, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, client.GocardlessAccountTypeMetadataKey)

		if err != nil {
			return fmt.Errorf("required metadata field com.gocardless.spec/account_type is missing")
		}

		if accountType != "checking" && accountType != "savings" {
			return fmt.Errorf("metadata field com.gocardless.spec/account_type must be checking or savings")

		}

	}

	if newExternalBankAccount.AccountNumber == nil {
		return fmt.Errorf("account number is required")
	}

	if newExternalBankAccount.SwiftBicCode == nil {
		return fmt.Errorf("swift bic code is required")
	}

	if newExternalBankAccount.Country == nil {
		return fmt.Errorf("country is required")
	}

	if newExternalBankAccount.IBAN == nil {
		return fmt.Errorf("IBAN is required")
	}

	return nil
}

func (p *Plugin) createBankAccount(ctx context.Context, ba models.BankAccount) (models.CreateBankAccountResponse, error) {
	err := validateExternalBankAccount(ba)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	creditor := ba.Metadata[client.GocardlessCreditorMetadataKey]
	customer := ba.Metadata[client.GocardlessCustomerMetadataKey]
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
