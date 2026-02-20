package stripe

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/stripe/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/stripe/stripe-go/v80"
)

var (
	legacyRootAccountReference = "root"
)

// we used to create a dummy ID for the root account which we don't want to pass to Stripe API clients
func resolveAccount(ref string) string {
	if ref == legacyRootAccountReference {
		return ""
	}
	return ref
}

type accountsState struct {
	RootCreated bool            `json:"root_created"`
	Timeline    client.Timeline `json:"timeline"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}
	}

	accounts := make([]connector.PSPAccount, 0, req.PageSize)
	if !oldState.RootCreated {
		// create a root account if this is the first time this is being run
		rootAccount, err := p.client.GetRootAccount()
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}
		pspAccount, err := ToPSPAccount(*rootAccount)
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}
		accounts = append(accounts, pspAccount)
		oldState.RootCreated = true
	}

	needed := req.PageSize - len(accounts)

	newState := oldState
	rawAccounts, timeline, hasMore, err := p.client.GetAccounts(ctx, oldState.Timeline, int64(needed))
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}
	newState.Timeline = timeline

	for _, acc := range rawAccounts {
		pspAccount, err := ToPSPAccount(*acc)
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}
		accounts = append(accounts, pspAccount)
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}
	return connector.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func ToPSPAccount(acc stripe.Account) (connector.PSPAccount, error) {
	raw, err := json.Marshal(acc)
	if err != nil {
		return connector.PSPAccount{}, err
	}

	metadata := make(map[string]string)
	for k, v := range acc.Metadata {
		metadata[k] = v
	}

	var displayName string
	if acc.Settings != nil && acc.Settings.Dashboard != nil {
		displayName = acc.Settings.Dashboard.DisplayName
	}

	defaultAsset := currency.FormatAsset(supportedCurrenciesWithDecimal, string(acc.DefaultCurrency))
	return connector.PSPAccount{
		Name:         &displayName,
		Reference:    acc.ID,
		CreatedAt:    time.Unix(acc.Created, 0).UTC(),
		DefaultAsset: &defaultAsset,
		Raw:          raw,
		Metadata:     metadata,
	}, nil
}
