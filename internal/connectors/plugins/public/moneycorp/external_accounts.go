package moneycorp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moneycorp/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type externalAccountsState struct {
	LastPage int `json:"last_page"`
	// Moneycorp does not allow us to sort by , but we can still
	// sort by ID created (which is incremental when creating accounts).
	LastCreatedAt time.Time `json:"last_created_at"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}

	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextExternalAccountsResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	newState := externalAccountsState{
		LastPage:      oldState.LastPage,
		LastCreatedAt: oldState.LastCreatedAt,
	}

	hasMore := false
	accounts := make([]models.PSPAccount, 0, req.PageSize)
	for page := oldState.LastPage; ; page++ {
		newState.LastPage = page
		pageSize := req.PageSize - len(accounts)

		pagedRecipients, err := p.client.GetRecipients(ctx, from.Reference, page, pageSize)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
		if len(pagedRecipients) == 0 {
			hasMore = false
			break
		}

		var lastCreatedAt time.Time
		accounts, lastCreatedAt, err = recipientToPSPAccounts(oldState.LastCreatedAt, accounts, pagedRecipients)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
		if len(accounts) == 0 {
			break
		}
		newState.LastCreatedAt = lastCreatedAt

		needMore := true
		needMore, hasMore = pagination.ShouldFetchMore(accounts, pagedRecipients, pageSize)
		if !needMore {
			break
		}
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: accounts,
		NewState:         payload,
		HasMore:          hasMore,
	}, nil
}

func recipientToPSPAccounts(
	lastCreatedAt time.Time,
	accounts []models.PSPAccount,
	pagedAccounts []*client.Recipient,
) ([]models.PSPAccount, time.Time, error) {
	var newCreatedAt time.Time
	for _, recipient := range pagedAccounts {
		createdAt, err := time.Parse("2006-01-02T15:04:05.999999999", recipient.Attributes.CreatedAt)
		if err != nil {
			return accounts, lastCreatedAt, fmt.Errorf("failed to parse transaction date: %v", err)
		}

		switch createdAt.Compare(lastCreatedAt) {
		case -1, 0:
			continue
		default:
		}

		raw, err := json.Marshal(recipient)
		if err != nil {
			return accounts, lastCreatedAt, err
		}

		newCreatedAt = createdAt
		accounts = append(accounts, models.PSPAccount{
			Reference: recipient.ID,
			// Moneycorp does not send the opening date of the account
			CreatedAt:    createdAt,
			Name:         &recipient.Attributes.BankAccountName,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, recipient.Attributes.BankAccountCurrency)),
			Raw:          raw,
		})
	}
	return accounts, newCreatedAt, nil
}
