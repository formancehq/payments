package client

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"
)

func (c *client) GetExternalAccounts(ctx context.Context, ownerID string, pageSize int, after string, before string) ([]GocardlessGenericAccount, Cursor, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_external_accounts")

	var nextCursor Cursor
	var bankAccounts []GocardlessGenericAccount
	var err error

	if ownerID[:2] == "CR" {
		bankAccounts, nextCursor, err = c.getCreditorExternalAccounts(ctx, ownerID, pageSize, after, before)
	}

	if ownerID[:2] == "CU" {
		bankAccounts, nextCursor, err = c.getCustomerExternalAccounts(ctx, ownerID, pageSize, after, before)
	}

	if err != nil {
		return []GocardlessGenericAccount{}, Cursor{}, err
	}

	return bankAccounts, nextCursor, nil
}

func (c *client) getCreditorExternalAccounts(ctx context.Context, creditor string, pageSize int, after string, before string) ([]GocardlessGenericAccount, Cursor, error) {

	accountsResponse, err := c.service.CreditorBankAccounts.List(ctx, gocardless.CreditorBankAccountListParams{
		Creditor: creditor,
		After:    after,
		Before:   before,
		Limit:    pageSize,
	})

	bankAccounts := make([]GocardlessGenericAccount, 0, pageSize)

	if err != nil {
		return []GocardlessGenericAccount{}, Cursor{}, err
	}

	for _, creditorBankAccount := range accountsResponse.CreditorBankAccounts {
		parsedTime, err := time.Parse(time.RFC3339Nano, creditorBankAccount.CreatedAt)
		if err != nil {
			return []GocardlessGenericAccount{}, Cursor{}, fmt.Errorf("failed to parse creation time: %w", err)
		}

		bankAccounts = append(bankAccounts, GocardlessGenericAccount{
			AccountHolderName: creditorBankAccount.AccountHolderName,
			ID:                creditorBankAccount.Id,
			CreatedAt:         parsedTime.Unix(),
			Currency:          creditorBankAccount.Currency,
			Metadata:          creditorBankAccount.Metadata,
		})
	}

	return bankAccounts, Cursor{}, nil
}

func (c *client) getCustomerExternalAccounts(ctx context.Context, customer string, pageSize int, after string, before string) ([]GocardlessGenericAccount, Cursor, error) {

	accountsResponse, err := c.service.CustomerBankAccounts.List(ctx, gocardless.CustomerBankAccountListParams{
		Customer: customer,
		After:    after,
		Before:   before,
		Limit:    pageSize,
	})

	var bankAccounts []GocardlessGenericAccount

	if err != nil {
		return []GocardlessGenericAccount{}, Cursor{}, err
	}

	for _, customerBankAccount := range accountsResponse.CustomerBankAccounts {
		parsedTime, err := time.Parse(time.RFC3339Nano, customerBankAccount.CreatedAt)
		if err != nil {
			return []GocardlessGenericAccount{}, Cursor{}, fmt.Errorf("failed to parse creation time: %w", err)
		}

		bankAccounts = append(bankAccounts, GocardlessGenericAccount{
			AccountHolderName: customerBankAccount.AccountHolderName,
			ID:                customerBankAccount.Id,
			CreatedAt:         parsedTime.Unix(),
			Currency:          customerBankAccount.Currency,
			Metadata:          customerBankAccount.Metadata,
		})
	}

	return bankAccounts, Cursor{}, nil
}
