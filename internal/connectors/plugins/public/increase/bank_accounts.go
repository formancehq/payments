package increase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createBankAccount(ctx context.Context, ba models.BankAccount) (models.CreateBankAccountResponse, error) {
	if err := p.validateBankAccountRequests(ba); err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	resp, err := p.client.CreateBankAccount(
		ctx,
		&client.BankAccountRequest{
			AccountNumber: *ba.AccountNumber,
			RoutingNumber: models.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseRoutingNumberMetadataKey),
			AccountHolder: models.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseAccountHolderMetadataKey),
			Description:   models.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseDescriptionMetadataKey),
		},
	)
	if err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	var account models.PSPAccount
	if resp != nil {
		raw, err := json.Marshal(resp)
		if err != nil {
			return models.CreateBankAccountResponse{}, err
		}
		createdTime, _ := time.Parse("2006-01-02T15:04:05.999-0700", resp.CreatedAt)
		account = models.PSPAccount{
			Reference: resp.ID,
			CreatedAt: createdTime,
			Metadata: map[string]string{
				"accountHolder": resp.AccountHolder,
				"accountNumber": resp.AccountNumber,
				"description":   resp.Description,
				"routingNumber": resp.RoutingNumber,
				"type":          resp.Type,
				"status":        resp.Status,
			},
			Raw: raw,
		}
	}

	return models.CreateBankAccountResponse{
		RelatedAccount: account,
	}, nil
}
