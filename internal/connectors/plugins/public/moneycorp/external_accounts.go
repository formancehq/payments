package moneycorp

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moneycorp/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type externalAccountsState struct {
	LastCreatedAt time.Time `json:"LastCreatedAt"`
	// LastProcessedIDs holds the references (recipient IDs) of ALL accounts
	// already emitted at exactly LastCreatedAt, accumulated across cycles while
	// the watermark second is unchanged and reset when it advances. Each cycle
	// rescans from page 0 and skips this whole set: a same-second group larger
	// than PageSize is walked across cycles without a drifting page cursor, and a
	// multi-row final page cannot oscillate (every already-emitted sibling is
	// skipped, not just one).
	//
	// Migration note: the old singular lastProcessedID and lastPage fields are
	// ignored. After deploy the watermark second is re-emitted once (idempotent —
	// storage upserts dedup it) and no recrawl occurs.
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
		LastCreatedAt:    oldState.LastCreatedAt,
		LastProcessedIDs: oldState.LastProcessedIDs,
	}

	needMore := false
	hasMore := false
	accounts := make([]models.PSPAccount, 0, req.PageSize)
	// Rescan from page 0 each cycle (no page cursor): the processed-ID set skips
	// every already-emitted sibling at the watermark second, so a same-second
	// group larger than PageSize is walked across cycles and a multi-row final
	// page cannot oscillate. There is no server-side time filter, so the skip is
	// applied client-side in recipientToPSPAccounts.
	for page := 0; ; page++ {
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
	}

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
