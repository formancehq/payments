package increase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/pkg/connectors/increase/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) createBankAccount(ctx context.Context, ba connector.BankAccount) (connector.CreateBankAccountResponse, error) {
	if err := p.validateBankAccountRequests(ba); err != nil {
		return connector.CreateBankAccountResponse{}, err
	}

	idempotencyKey := p.generateIdempotencyKey(ba.ID.String())
	resp, err := p.client.CreateBankAccount(
		ctx,
		&client.BankAccountRequest{
			AccountNumber: *ba.AccountNumber,
			RoutingNumber: connector.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseRoutingNumberMetadataKey),
			AccountHolder: connector.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseAccountHolderMetadataKey),
			Description:   connector.ExtractNamespacedMetadata(ba.Metadata, client.IncreaseDescriptionMetadataKey),
		},
		idempotencyKey,
	)
	if err != nil {
		return connector.CreateBankAccountResponse{}, err
	}

	var account connector.PSPAccount
	if resp != nil {
		raw, err := json.Marshal(resp)
		if err != nil {
			return connector.CreateBankAccountResponse{}, err
		}
		createdTime, err := time.Parse(time.RFC3339, resp.CreatedAt)
		if err != nil {
			return connector.CreateBankAccountResponse{}, fmt.Errorf("failed to parse creation time: %w", err)
		}
		account = connector.PSPAccount{
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

	return connector.CreateBankAccountResponse{
		RelatedAccount: account,
	}, nil
}
