package atlar

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func validateExternalBankAccount(newExternalBankAccount models.BankAccount) error {
	_, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, "owner/name")
	if err != nil {
		return fmt.Errorf("required metadata field %sowner/name is missing", atlarMetadataSpecNamespace)
	}

	ownerType, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, "owner/type")
	if err != nil {
		return fmt.Errorf("required metadata field %sowner/type is missing", atlarMetadataSpecNamespace)
	}

	if ownerType != "INDIVIDUAL" && ownerType != "COMPANY" {
		return fmt.Errorf("metadata field %sowner/type needs to be one of [ INDIVIDUAL COMPANY ]", atlarMetadataSpecNamespace)
	}

	return nil
}

func (p *Plugin) createBankAccount(ctx context.Context, ba models.BankAccount) (models.CreateBankAccountResponse, error) {
	err := validateExternalBankAccount(ba)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	resp, err := p.client.PostV1CounterParties(ctx, ba)
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
