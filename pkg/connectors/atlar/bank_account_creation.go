package atlar

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/pkg/connector"
)

func validateExternalBankAccount(newExternalBankAccount connector.BankAccount) error {
	_, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, "owner/name")
	if err != nil {
		return connector.NewWrappedError(
			fmt.Errorf("required metadata field %sowner/name is missing", atlarMetadataSpecNamespace),
			connector.ErrInvalidRequest,
		)
	}

	ownerType, err := extractNamespacedMetadata(newExternalBankAccount.Metadata, "owner/type")
	if err != nil {
		return connector.NewWrappedError(
			fmt.Errorf("required metadata field %sowner/type is missing", atlarMetadataSpecNamespace),
			connector.ErrInvalidRequest,
		)
	}

	if ownerType != "INDIVIDUAL" && ownerType != "COMPANY" {
		return connector.NewWrappedError(
			fmt.Errorf("metadata field %sowner/type needs to be one of [ INDIVIDUAL COMPANY ]", atlarMetadataSpecNamespace),
			connector.ErrInvalidRequest,
		)
	}

	return nil
}

func (p *Plugin) createBankAccount(ctx context.Context, ba connector.BankAccount) (connector.CreateBankAccountResponse, error) {
	err := validateExternalBankAccount(ba)
	if err != nil {
		return connector.CreateBankAccountResponse{}, err
	}

	resp, err := p.client.PostV1CounterParties(ctx, ba)
	if err != nil {
		return connector.CreateBankAccountResponse{}, err
	}

	if resp == nil {
		return connector.CreateBankAccountResponse{},
			connector.NewWrappedError(
				fmt.Errorf("PostV1CounterParties: unexpected empty response"),
				connector.ErrFailedAccountCreation,
			)
	}

	newAccount, err := externalAccountFromAtlarData(resp.Payload.ExternalAccounts[0], resp.Payload)
	if err != nil {
		return connector.CreateBankAccountResponse{}, err
	}

	return connector.CreateBankAccountResponse{
		RelatedAccount: newAccount,
	}, nil

}
