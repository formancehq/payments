package increase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/pkg/connectors/increase/client"
	"github.com/formancehq/payments/pkg/connector"
)

type externalAccountsState struct {
	NextCursor string `json:"next_cursor"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req connector.FetchNextExternalAccountsRequest) (connector.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}
	}

	accounts := make([]connector.PSPAccount, 0, req.PageSize)
	hasMore := false
	pagedRecipients, nextCursor, err := p.client.GetExternalAccounts(ctx, req.PageSize, oldState.NextCursor)
	if err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	accounts, err = p.fillExternalAccounts(pagedRecipients, accounts, req.PageSize)
	if err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	hasMore = nextCursor != ""

	newState := externalAccountsState{
		NextCursor: nextCursor,
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	return connector.FetchNextExternalAccountsResponse{
		ExternalAccounts: accounts,
		NewState:         payload,
		HasMore:          hasMore,
	}, nil
}

func (p *Plugin) fillExternalAccounts(
	pagedAccounts []*client.ExternalAccount,
	accounts []connector.PSPAccount,
	pageSize int,
) ([]connector.PSPAccount, error) {
	for _, account := range pagedAccounts {
		if len(accounts) >= pageSize {
			break
		}

		mappedAccounts, err := p.mapExternalAccount(account)
		if err != nil {
			return nil, fmt.Errorf("failed to map external account: %w", err)
		}
		accounts = append(accounts, *mappedAccounts)
	}

	return accounts, nil
}

func (p *Plugin) mapExternalAccount(
	account *client.ExternalAccount,
) (*connector.PSPAccount, error) {
	createdTime, err := time.Parse(time.RFC3339, account.CreatedAt)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(account)
	if err != nil {
		return nil, err
	}

	pspAccount := connector.PSPAccount{
		Reference: account.ID,
		CreatedAt: createdTime,
		Name:      &account.Description,
		Raw:       raw,
		Metadata: map[string]string{
			client.IncreaseTypeMetadataKey:          account.Type,
			client.IncreaseAccountHolderMetadataKey: account.AccountHolder,
			client.IncreaseAccountNumberMetadataKey: account.AccountNumber,
			client.IncreaseStatusMetadataKey:        account.Status,
			client.IncreaseDescriptionMetadataKey:   account.Description,
			client.IncreaseRoutingNumberMetadataKey: account.RoutingNumber,
		},
	}

	return &pspAccount, nil
}
