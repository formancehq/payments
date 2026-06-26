package moneycorp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moneycorp/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type externalAccountsState struct {
	LastPage      int       `json:"lastPage"`
	LastCreatedAt time.Time `json:"LastCreatedAt"`
	// LastProcessedID is the reference (recipient ID) of the last account emitted
	// at exactly LastCreatedAt, so the inclusive (>=) watermark filter excludes
	// only that already-processed row while keeping distinct same-timestamp ones.
	LastProcessedID string `json:"lastProcessedID"`
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
		LastPage:        oldState.LastPage,
		LastCreatedAt:   oldState.LastCreatedAt,
		LastProcessedID: oldState.LastProcessedID,
	}

	needMore := false
	hasMore := false
	accounts := make([]models.PSPAccount, 0, req.PageSize)
	for page := oldState.LastPage; ; page++ {
		newState.LastPage = page

		pagedRecipients, err := p.client.GetRecipients(ctx, from.Reference, page, req.PageSize)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		accounts, err = recipientToPSPAccounts(oldState.LastCreatedAt, oldState.LastProcessedID, accounts, pagedRecipients)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(accounts, pagedRecipients, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		accounts = accounts[:req.PageSize]
	}

	if len(accounts) > 0 {
		newState.LastCreatedAt = accounts[len(accounts)-1].CreatedAt
		newState.LastProcessedID = accounts[len(accounts)-1].Reference
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
	lastProcessedID string,
	accounts []models.PSPAccount,
	pagedAccounts []*client.Recipient,
) ([]models.PSPAccount, error) {
	for _, recipient := range pagedAccounts {
		createdAt, err := time.Parse("2006-01-02T15:04:05.999999999", recipient.Attributes.CreatedAt)
		if err != nil {
			return accounts, fmt.Errorf("failed to parse transaction date: %v", err)
		}

		// Inclusive watermark: skip recipients strictly before it, and the single
		// already-processed recipient at exactly the watermark. Distinct recipients
		// sharing that timestamp are kept (M-CON2).
		cmp := createdAt.Compare(lastCreatedAt)
		if cmp < 0 || (cmp == 0 && recipient.ID == lastProcessedID) {
			continue
		}

		raw, err := json.Marshal(recipient)
		if err != nil {
			return accounts, err
		}

		accounts = append(accounts, models.PSPAccount{
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
