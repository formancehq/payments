package wise

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/pkg/connectors/wise/client"
	"github.com/formancehq/payments/pkg/connector"
)

type accountsState struct {
	// Accounts are ordered by their ID
	LastAccountID uint64 `json:"lastAccountID"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}
	}

	var from client.Profile
	if req.FromPayload == nil {
		return connector.FetchNextAccountsResponse{}, connector.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	newState := accountsState{
		LastAccountID: oldState.LastAccountID,
	}

	var accounts []connector.PSPAccount
	hasMore := false
	// Wise balances are considered as accounts on our side.
	balances, err := p.client.GetBalances(ctx, from.ID)
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	for _, balance := range balances {
		if oldState.LastAccountID != 0 && balance.ID <= oldState.LastAccountID {
			continue
		}

		raw, err := json.Marshal(balance)
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}

		accounts = append(accounts, connector.PSPAccount{
			Reference:    strconv.FormatUint(balance.ID, 10),
			CreatedAt:    balance.CreationTime,
			Name:         &balance.Name,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, balance.Amount.Currency)),
			Metadata: map[string]string{
				metadataProfileIDKey: strconv.FormatUint(from.ID, 10),
			},
			Raw: raw,
		})

		newState.LastAccountID = balance.ID

		if len(accounts) >= req.PageSize {
			hasMore = true
			break
		}
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
