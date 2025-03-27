package atlar

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func validateExternalBankAccount(newExternalBankAccount models.BankAccount) error {
	_, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, "owner/name")
	if err != nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("required metadata field %sowner/name is missing", atlarMetadataSpecNamespace),
			models.ErrInvalidRequest,
		)
	}

	ownerType, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, "owner/type")
	if err != nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("required metadata field %sowner/type is missing", atlarMetadataSpecNamespace),
			models.ErrInvalidRequest,
		)
	}

	if ownerType != "INDIVIDUAL" && ownerType != "COMPANY" {
		return errorsutils.NewWrappedError(
			fmt.Errorf("metadata field %sowner/type needs to be one of [ INDIVIDUAL COMPANY ]", atlarMetadataSpecNamespace),
			models.ErrInvalidRequest,
		)
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
		return models.CreateBankAccountResponse{},
			errorsutils.NewWrappedError(
				fmt.Errorf("PostV1CounterParties: unexpected empty response"),
				models.ErrFailedAccountCreation,
			)
	}

	newAccount, err := externalAccountFromAtlarData(resp.Payload.ExternalAccounts[0], resp.Payload)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	return models.CreateBankAccountResponse{
		RelatedAccount: newAccount,
	}, nil

}
