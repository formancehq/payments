package increase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) createBankAccount(ctx context.Context, ba models.BankAccount) (models.CreateBankAccountResponse, error) {
	if err := p.validateBankAccountRequests(ba); err != nil {
		return models.CreateBankAccountResponse{}, err
	}

	idempotencyKey := p.generateIdempotencyKey(ba.ID.String())
	resp, err := p.client.CreateBankAccount(
		ctx,
		&client.BankAccountRequest{
			AccountNumber: *ba.AccountNumber,
			RoutingNumber: models.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseRoutingNumberMetadataKey),
			AccountHolder: models.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseAccountHolderMetadataKey),
			Description:   models.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseDescriptionMetadataKey),
		},
		idempotencyKey,
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
		createdTime, err := time.Parse(time.RFC3339, resp.CreatedAt)
		if err != nil {
			return models.CreateBankAccountResponse{}, fmt.Errorf("failed to parse creation time: %w", err)
		}
		account = models.PSPAccount{
			Reference: resp.ID,
			CreatedAt: createdTime,
			Name:      &resp.Description,
			Metadata: map[string]string{
				client.IncreaseAccountHolderMetadataKey: resp.AccountHolder,
				client.IncreaseAccountNumberMetadataKey: resp.AccountNumber,
				client.IncreaseDescriptionMetadataKey:   resp.Description,
				client.IncreaseRoutingNumberMetadataKey: resp.RoutingNumber,
				client.IncreaseTypeMetadataKey:          resp.Type,
				client.IncreaseStatusMetadataKey:        resp.Status,
			},
			Raw: raw,
		}
	}

	return models.CreateBankAccountResponse{
		RelatedAccount: account,
	}, nil
}
