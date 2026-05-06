package routable

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	state, err := decodePageState(req.State)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	resp, err := p.client.ListAccounts(ctx, state.nextPage(), req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("listing accounts (page=%d): %w", state.nextPage(), err)
	}

	accounts := make([]models.PSPAccount, 0, len(resp.Results))
	for _, a := range resp.Results {
		account, err := p.settingsAccountToPSPAccount(a)
		if err != nil {
			p.logger.Infof("skipping settings account %s: %v", a.ID, err)
			continue
		}
		accounts = append(accounts, account)
	}

	newState := pageState{Page: state.nextPage() + 1}
	if !resp.Links.HasMore() {
		newState.Page = 1 // restart on next cycle
	}
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("marshaling state: %w", err)
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  resp.Links.HasMore(),
	}, nil
}

// settingsAccountToPSPAccount maps a Routable settings account onto a
// Formance internal PSPAccount. Routable carries the currency on
// type_details (and historically defaulted to USD when absent); we surface
// it as DefaultAsset only when we recognize the code.
func (p *Plugin) settingsAccountToPSPAccount(a client.Account) (models.PSPAccount, error) {
	raw, err := json.Marshal(a)
	if err != nil {
		return models.PSPAccount{}, fmt.Errorf("marshaling raw: %w", err)
	}
	out := models.PSPAccount{
		Reference: a.ID,
		CreatedAt: a.CreatedAt,
		Name:      pointerOrNil(a.Name),
		Metadata:  settingsAccountMetadata(a),
		Raw:       raw,
	}
	if asset := formatAsset(a.CurrencyCode); asset != "" {
		out.DefaultAsset = &asset
	}
	return out, nil
}

func pointerOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func decodePageState(raw json.RawMessage) (pageState, error) {
	var s pageState
	if len(raw) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("decoding state: %w", err)
	}
	return s, nil
}
