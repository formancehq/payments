package client

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"
)

func (c *client) CreateCreditorBankAccount(ctx context.Context, creditor string, ba models.BankAccount) (GocardlessGenericAccount, error) {

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_creditor_bank_account")

	setAsDefaultPayoutAccount := false

	currency := models.ExtractNamespacedMetadata(ba.Metadata, GocardlessCurrencyMetadataKey)
	accountType := models.ExtractNamespacedMetadata(ba.Metadata, GocardlessAccountTypeMetadataKey)

	payload := gocardless.CreditorBankAccountCreateParams{
		AccountHolderName:         ba.Name,
		AccountNumber:             *ba.AccountNumber,
		AccountType:               accountType,
		CountryCode:               *ba.Country,
		Currency:                  currency,
		Links:                     gocardless.CreditorBankAccountCreateParamsLinks{Creditor: creditor},
		SetAsDefaultPayoutAccount: setAsDefaultPayoutAccount,
	}

	if *ba.Country == "US" {
		payload.BankCode = *ba.SwiftBicCode
	} else {
		payload.BranchCode = *ba.SwiftBicCode
	}

	bankAccount, err := c.service.CreateGocardlessCreditorBankAccount(ctx, payload)

	if err != nil {
		return GocardlessGenericAccount{}, err
	}

	parsedTime, err := time.Parse(time.RFC3339Nano, bankAccount.CreatedAt)

	if err != nil {
		return GocardlessGenericAccount{}, fmt.Errorf("failed to parse creation time: %w", err)
	}

	return GocardlessGenericAccount{
		ID:                bankAccount.Id,
		CreatedAt:         parsedTime,
		AccountHolderName: bankAccount.AccountHolderName,
		Metadata:          bankAccount.Metadata,
		Currency:          bankAccount.Currency,
		AccountType:       bankAccount.AccountType,
	}, nil

}

func (c *client) CreateCustomerBankAccount(ctx context.Context, customer string, ba models.BankAccount) (GocardlessGenericAccount, error) {

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_customer_bank_account")

	currency := models.ExtractNamespacedMetadata(ba.Metadata, GocardlessCurrencyMetadataKey)
	accountType := models.ExtractNamespacedMetadata(ba.Metadata, GocardlessAccountTypeMetadataKey)

	payload := gocardless.CustomerBankAccountCreateParams{
		AccountHolderName: ba.Name,
		AccountNumber:     *ba.AccountNumber,
		AccountType:       accountType,
		CountryCode:       *ba.Country,
		Currency:          currency,
		Links:             gocardless.CustomerBankAccountCreateParamsLinks{Customer: customer},
	}

	if *ba.Country == "US" {
		payload.BankCode = *ba.SwiftBicCode
	} else {
		payload.BranchCode = *ba.SwiftBicCode
	}

	bankAccount, err := c.service.CreateGocardlessCustomerBankAccount(ctx, payload)

	if err != nil {
		return GocardlessGenericAccount{}, err
	}

	parsedTime, err := time.Parse(time.RFC3339Nano, bankAccount.CreatedAt)
	if err != nil {
		return GocardlessGenericAccount{}, fmt.Errorf("failed to parse creation time: %w", err)
	}

	return GocardlessGenericAccount{
		ID:                bankAccount.Id,
		CreatedAt:         parsedTime,
		AccountHolderName: bankAccount.AccountHolderName,
		Metadata:          bankAccount.Metadata,
		Currency:          bankAccount.Currency,
		AccountType:       bankAccount.AccountType,
	}, nil

}
