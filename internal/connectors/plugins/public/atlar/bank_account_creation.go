package atlar

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/atlar/client"
	"github.com/formancehq/payments/internal/models"
)

func validateExternalBankAccount(newExternalBankAccount *models.BankAccount) error {
	_, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, "owner/name")
	if err != nil {
		return fmt.Errorf("required metadata field %sowner/name is missing: %w", atlarMetadataSpecNamespace, models.ErrInvalidRequest)
	}

	ownerType, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, "owner/type")
	if err != nil {
		return fmt.Errorf("required metadata field %sowner/type is missing: %w", atlarMetadataSpecNamespace, models.ErrInvalidRequest)
	}

	if ownerType != "INDIVIDUAL" && ownerType != "COMPANY" {
		return fmt.Errorf("metadata field %sowner/type needs to be one of [ INDIVIDUAL COMPANY ]: %w", atlarMetadataSpecNamespace, models.ErrInvalidRequest)
	}

	return nil
}

func validateCounterParty(newCounterParty *models.PSPCounterParty) error {
	if newCounterParty.Name == "" {
		return fmt.Errorf("counter party name is required: %w", models.ErrInvalidRequest)
	}

	ownerType, err := extractNamespacedMetadata(newCounterParty.Metadata, "owner/type")
	if err != nil {
		return fmt.Errorf("required metadata field %sowner/type is missing: %w", atlarMetadataSpecNamespace, models.ErrInvalidRequest)
	}

	if ownerType != "INDIVIDUAL" && ownerType != "COMPANY" {
		return fmt.Errorf("metadata field %sowner/type needs to be one of [ INDIVIDUAL COMPANY ]: %w", atlarMetadataSpecNamespace, models.ErrInvalidRequest)
	}

	return nil
}

func (p *Plugin) createBankAccountFromBankAccount(ctx context.Context, ba *models.BankAccount) (models.CreateBankAccountResponse, error) {
	err := validateExternalBankAccount(ba)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	createCounterpartyRequest := client.ToAtlarCreateCounterpartyRequestFromBankAccount(ba)
	resp, err := p.client.PostV1CounterParties(ctx, createCounterpartyRequest)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	if resp == nil {
		return models.CreateBankAccountResponse{}, fmt.Errorf("unexpected empty response: %w", models.ErrFailedAccountCreation)
	}

	newAccount, err := externalAccountFromAtlarData(resp.Payload.ExternalAccounts[0], resp.Payload)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	return models.CreateBankAccountResponse{
		RelatedAccount: newAccount,
	}, nil

}

func (p *Plugin) createBankAccountFromCounterParty(ctx context.Context, cp *models.PSPCounterParty) (models.CreateBankAccountResponse, error) {
	err := validateCounterParty(cp)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	createCounterpartyRequest := client.ToAtlarCreateCounterpartyRequestFromCounterParty(cp)
	resp, err := p.client.PostV1CounterParties(ctx, createCounterpartyRequest)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	if resp == nil {
		return models.CreateBankAccountResponse{}, fmt.Errorf("unexpected empty response: %w", models.ErrFailedAccountCreation)
	}

	newAccount, err := externalAccountFromAtlarData(resp.Payload.ExternalAccounts[0], resp.Payload)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	return models.CreateBankAccountResponse{
		RelatedAccount: newAccount,
	}, nil
}
