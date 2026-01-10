package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/generic/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (p *Plugin) createBankAccount(ctx context.Context, ba models.BankAccount) (models.CreateBankAccountResponse, error) {
	if err := p.validateBankAccountRequest(ba); err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	req := &client.BankAccountRequest{
		Name:          ba.Name,
		AccountNumber: ba.AccountNumber,
		IBAN:          ba.IBAN,
		SwiftBicCode:  ba.SwiftBicCode,
		Country:       ba.Country,
		Metadata:      ba.Metadata,
	}

	resp, err := p.client.CreateBankAccount(ctx, req)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	return bankAccountResponseToAccount(resp)
}

func (p *Plugin) validateBankAccountRequest(ba models.BankAccount) error {
	if ba.Name == "" {
		return errorsutils.NewWrappedError(
			fmt.Errorf("name is required"),
			models.ErrInvalidRequest,
		)
	}

	// At least one account identifier should be provided
	if (ba.AccountNumber == nil || *ba.AccountNumber == "") && (ba.IBAN == nil || *ba.IBAN == "") {
		return errorsutils.NewWrappedError(
			fmt.Errorf("either account number or IBAN is required"),
			models.ErrInvalidRequest,
		)
	}

	return nil
}

func bankAccountResponseToAccount(resp *client.BankAccountResponse) (models.CreateBankAccountResponse, error) {
	createdAt, err := time.Parse(time.RFC3339, resp.CreatedAt)
	if err != nil {
		return models.CreateBankAccountResponse{}, fmt.Errorf("failed to parse created at: %w", err)
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		return models.CreateBankAccountResponse{}, fmt.Errorf("failed to marshal raw response: %w", err)
	}

	metadata := resp.Metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}

	// Add bank account details to metadata for later retrieval
	if resp.AccountNumber != nil {
		metadata[models.AccountAccountNumberMetadataKey] = *resp.AccountNumber
	}
	if resp.IBAN != nil {
		metadata[models.AccountIBANMetadataKey] = *resp.IBAN
	}
	if resp.SwiftBicCode != nil {
		metadata[models.AccountSwiftBicCodeMetadataKey] = *resp.SwiftBicCode
	}
	if resp.Country != nil {
		metadata[models.AccountBankAccountCountryMetadataKey] = *resp.Country
	}

	return models.CreateBankAccountResponse{
		RelatedAccount: models.PSPAccount{
			Reference: resp.Id,
			CreatedAt: createdAt,
			Name:      &resp.Name,
			Raw:       raw,
			Metadata:  metadata,
		},
	}, nil
}
