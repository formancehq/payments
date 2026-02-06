package moneycorp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/moneycorp/client"
	"github.com/formancehq/payments/pkg/connector"
	
)

type externalAccountsState struct {
	LastPage      int       `json:"lastPage"`
	LastCreatedAt time.Time `json:"LastCreatedAt"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req connector.FetchNextExternalAccountsRequest) (connector.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}
	}

	var from connector.PSPAccount
	if req.FromPayload == nil {
		return connector.FetchNextExternalAccountsResponse{}, connector.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	newState := externalAccountsState{
		LastPage:      oldState.LastPage,
		LastCreatedAt: oldState.LastCreatedAt,
	}

	needMore := false
	hasMore := false
	accounts := make([]connector.PSPAccount, 0, req.PageSize)
	for page := oldState.LastPage; ; page++ {
		newState.LastPage = page

		pagedRecipients, err := p.client.GetRecipients(ctx, from.Reference, page, req.PageSize)
		if err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}

		accounts, err = recipientToPSPAccounts(oldState.LastCreatedAt, accounts, pagedRecipients)
		if err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}

		needMore, hasMore = connector.ShouldFetchMore(accounts, pagedRecipients, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		accounts = accounts[:req.PageSize]
	}

	if len(accounts) > 0 {
		newState.LastCreatedAt = accounts[len(accounts)-1].CreatedAt
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

func recipientToPSPAccounts(
	lastCreatedAt time.Time,
	accounts []connector.PSPAccount,
	pagedAccounts []*client.Recipient,
) ([]connector.PSPAccount, error) {
	for _, recipient := range pagedAccounts {
		createdAt, err := time.Parse("2006-01-02T15:04:05.999999999", recipient.Attributes.CreatedAt)
		if err != nil {
			return accounts, fmt.Errorf("failed to parse transaction date: %v", err)
		}

		switch createdAt.Compare(lastCreatedAt) {
		case -1, 0:
			continue
		default:
		}

		raw, err := json.Marshal(recipient)
		if err != nil {
			return accounts, err
		}

		accounts = append(accounts, connector.PSPAccount{
			Reference: recipient.ID,
			// Moneycorp does not send the opening date of the account
			CreatedAt:    createdAt,
			Name:         &recipient.Attributes.BankAccountName,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, recipient.Attributes.BankAccountCurrency)),
			Raw:          raw,
		})
	}
	return accounts, nil
}
