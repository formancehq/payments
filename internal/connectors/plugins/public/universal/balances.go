package universal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
)

// balancesState walks the account list (contract scopes balances per
// account; there is no global balances list). AccountLookup is preferred;
// the /v1/accounts fallback is for installations without it.
type balancesState struct {
	NextAccountIdx int `json:"nextAccountIdx"`
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	declared, ok := p.declaredSet()
	if !ok {
		return models.FetchNextBalancesResponse{}, plugins.ErrNotYetInstalled
	}
	if err := declared.require(models.CAPABILITY_FETCH_BALANCES); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	var st balancesState
	if len(req.State) > 0 {
		if err := json.Unmarshal(req.State, &st); err != nil {
			return models.FetchNextBalancesResponse{}, err
		}
	}

	accounts, err := p.listAccountsForBalances(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("listing accounts for balances: %w", err)
	}

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = PAGE_SIZE
	}

	balances := make([]models.PSPBalance, 0, pageSize)
	idx := st.NextAccountIdx
	for ; idx < len(accounts) && len(balances) < pageSize; idx++ {
		res, err := p.client.GetBalances(ctx, accounts[idx].Reference)
		if err != nil {
			return models.FetchNextBalancesResponse{}, fmt.Errorf("getting balances for %s: %w", accounts[idx].Reference, err)
		}
		for _, b := range res.Items {
			pb, err := mappers.BalanceToPSPBalance(b)
			if err != nil {
				return models.FetchNextBalancesResponse{}, err
			}
			balances = append(balances, pb)
		}
	}

	hasMore := idx < len(accounts)
	st.NextAccountIdx = idx
	if !hasMore {
		st.NextAccountIdx = 0
	}

	newState, err := json.Marshal(st)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	return models.FetchNextBalancesResponse{Balances: balances, NewState: newState, HasMore: hasMore}, nil
}

// listAccountsForBalances prefers the engine-injected AccountLookup
// (durable across pods); falls back to /v1/accounts page-1 for tests and
// small installations.
func (p *Plugin) listAccountsForBalances(ctx context.Context) ([]models.PSPAccount, error) {
	if p.accountLookup != nil {
		return p.accountLookup.ListAccountsByConnector(ctx)
	}
	page, err := p.client.ListAccounts(ctx, client.Pagination{PageSize: PAGE_SIZE})
	if err != nil {
		return nil, err
	}
	out := make([]models.PSPAccount, 0, len(page.Items))
	for _, a := range page.Items {
		pa, err := mappers.AccountToPSPAccount(a)
		if err != nil {
			return nil, err
		}
		out = append(out, pa)
	}
	return out, nil
}
