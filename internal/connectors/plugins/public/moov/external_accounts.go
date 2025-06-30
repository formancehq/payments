package moov

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {

	var from moov.Account

	if req.FromPayload != nil {
		if err := json.Unmarshal(req.FromPayload, &from); err != nil {
			return models.FetchNextExternalAccountsResponse{}, fmt.Errorf("failed to unmarshal from payload: %w", err)
		}
	}

	bankAccounts, err := p.client.GetExternalAccounts(ctx, from.AccountID)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	externalAccounts, err := p.fillExternalAccounts(from.AccountID, bankAccounts)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: externalAccounts,
		HasMore:          false,
	}, nil
}

func (p *Plugin) fillExternalAccounts(accountID string, bankAccounts []moov.BankAccount) ([]models.PSPAccount, error) {
	externalAccounts := make([]models.PSPAccount, 0, len(bankAccounts))
	for _, bankAccount := range bankAccounts {

		raw, err := json.Marshal(bankAccount)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal bank account: %w", err)
		}

		metadata := map[string]string{
			client.MoovBankNameMetadataKey:              bankAccount.BankName,
			client.MoovHolderTypeMetadataKey:            string(bankAccount.HolderType),
			client.MoovBankAccountTypeMetadataKey:       string(bankAccount.BankAccountType),
			client.MoovRoutingNumberMetadataKey:         bankAccount.RoutingNumber,
			client.MoovLastFourAccountNumberMetadataKey: bankAccount.LastFourAccountNumber,
			client.MoovUpdateOnMetadataKey:              bankAccount.UpdatedOn.Format(time.RFC3339),
			client.MoovStatusReasonMetadataKey:          string(bankAccount.StatusReason),
			client.MoovStatusMetadataKey:                string(bankAccount.Status),
			client.MoovFingerprintMetadataKey:           bankAccount.Fingerprint,
			client.MoovAccountIDMetadataKey:             accountID,
		}

		if bankAccount.ExceptionDetails != nil {
			metadata[client.MoovExceptionDetailsDescriptionMetadataKey] = bankAccount.ExceptionDetails.Description

			if bankAccount.ExceptionDetails.AchReturnCode != nil {
				metadata[client.MoovExceptionDetailsAchReturnCodeMetadataKey] = string(*bankAccount.ExceptionDetails.AchReturnCode)
			}

			if bankAccount.ExceptionDetails.RTPRejectionCode != nil {
				metadata[client.MoovExceptionDetailsRTPRejectionCodeMetadataKey] = string(*bankAccount.ExceptionDetails.RTPRejectionCode)
			}
		}

		externalAccounts = append(externalAccounts, models.PSPAccount{
			Reference: bankAccount.BankAccountID,
			CreatedAt: time.Now(),
			Name:      &bankAccount.HolderName,
			Raw:       raw,
			Metadata:  metadata,
		})
	}
	return externalAccounts, nil
}
