package moneycorp

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/formancehq/payments/ce/plugins/moneycorp/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type externalAccountsState struct {
	// LastPage is a monotonic forward cursor. This endpoint has NO server-side
	// time filter, so resuming the scan here (re-reading only the last page, not
	// the whole history each poll) avoids a full historical rescan on every sync.
	LastPage      int       `json:"lastPage"`
	LastCreatedAt time.Time `json:"LastCreatedAt"`
	// LastProcessedIDs holds the references (recipient IDs) of ALL accounts
	// already emitted at exactly LastCreatedAt, accumulated while the watermark
	// second is unchanged and reset when it advances. It dedups same-second rows
	// on the re-read page, so a multi-row boundary settles to empty instead of
	// oscillating (a single LastProcessedID could exclude only one of several
	// tied rows).
	//
	// Migration note: the old singular lastProcessedID is ignored. After deploy
	// the watermark second is re-emitted once (idempotent — storage upserts dedup
	// it) and no recrawl occurs.
	//
	// Precision: comparison and the ID set use the exact timestamp the API
	// returns (full precision, as in the qonto reference), not a truncated
	// second; "same-second" above is shorthand because these PSPs report
	// timestamps at second granularity, so equal values represent the same second.
	LastProcessedIDs []string `json:"lastProcessedIDs"`
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
		LastPage:         oldState.LastPage,
		LastCreatedAt:    oldState.LastCreatedAt,
		LastProcessedIDs: oldState.LastProcessedIDs,
	}

	needMore := false
	hasMore := false
	accounts := make([]models.PSPAccount, 0, req.PageSize)
	// No server-side time filter: resume the monotonic forward scan at the
	// persisted page (re-reading only the last page, not the whole history) and
	// skip already-emitted same-second rows via the ID set (client-side, in
	// recipientToPSPAccounts), so a multi-row final page cannot oscillate.
	page := oldState.LastPage
	for {
		pagedRecipients, err := p.client.GetRecipients(ctx, from.Reference, page, req.PageSize)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		accounts, err = recipientToPSPAccounts(oldState.LastCreatedAt, oldState.LastProcessedIDs, accounts, pagedRecipients)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(accounts, pagedRecipients, req.PageSize)
		if !needMore || !hasMore {
			break
		}
		page++
	}
	newState.LastPage = page

	if len(accounts) > 0 {
		last := accounts[len(accounts)-1].CreatedAt

		// Collect the references emitted at exactly the new watermark second.
		idsAtWatermark := make([]string, 0)
		for i := range accounts {
			if accounts[i].CreatedAt.Equal(last) {
				idsAtWatermark = append(idsAtWatermark, accounts[i].Reference)
			}
		}

		// Accumulate the processed-ID set while still inside the same watermark
		// second; reset it when the watermark advances to a newer second.
		if last.Equal(oldState.LastCreatedAt) {
			newState.LastProcessedIDs = append(oldState.LastProcessedIDs, idsAtWatermark...)
		} else {
			newState.LastProcessedIDs = idsAtWatermark
		}
		newState.LastCreatedAt = last
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
	lastProcessedIDs []string,
	accounts []models.PSPAccount,
	pagedAccounts []*client.Recipient,
) ([]models.PSPAccount, error) {
	for _, recipient := range pagedAccounts {
		createdAt, err := time.Parse("2006-01-02T15:04:05.999999999", recipient.Attributes.CreatedAt)
		if err != nil {
			return accounts, fmt.Errorf("failed to parse transaction date: %v", err)
		}

		// Inclusive watermark: skip recipients strictly before it, and any
		// already-emitted recipient at exactly the watermark second. Distinct
		// recipients sharing that timestamp are kept (M-CON2).
		cmp := createdAt.Compare(lastCreatedAt)
		if cmp < 0 || (cmp == 0 && slices.Contains(lastProcessedIDs, recipient.ID)) {
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
