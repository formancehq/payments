package fireblocks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/ee/plugins/fireblocks/client"
	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	NextCursor string `json:"nextCursor"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	resp, err := p.client.GetVaultAccountsPaged(ctx, oldState.NextCursor, int(req.PageSize))
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, len(resp.Accounts))
	for _, account := range resp.Accounts {
		raw, err := json.Marshal(account)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		createdAt := time.Now()
		if account.CreationDate > 0 {
			createdAt = time.UnixMilli(account.CreationDate)
		}

		accounts = append(accounts, models.PSPAccount{
			Reference: account.ID,
			CreatedAt: createdAt,
			Name:      &account.Name,
			Metadata:  buildAccountMetadata(account),
			Raw:       raw,
		})
	}

	newState := accountsState{
		NextCursor: resp.Paging.After,
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	hasMore := resp.Paging.After != ""

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

// buildAccountMetadata surfaces vault-level context. Each key is emitted only
// when the source field is non-zero (non-empty string, true bool); nil is
// returned when nothing applies.
func buildAccountMetadata(a client.VaultAccount) map[string]string {
	m := map[string]string{}
	if a.CustomerRefID != "" {
		m[MetadataPrefix+"customer_ref_id"] = a.CustomerRefID
	}
	if a.HiddenOnUI {
		m[MetadataPrefix+"hidden_on_ui"] = "true"
	}
	if a.AutoFuel {
		m[MetadataPrefix+"auto_fuel"] = "true"
	}
	if len(m) == 0 {
		return nil
	}
	return m
}
