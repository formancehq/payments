package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/stripe/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/pkg/errors"
)

type externalAccountsState struct {
	Timeline client.Timeline `json:"timeline"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req connector.FetchNextExternalAccountsRequest) (connector.FetchNextExternalAccountsResponse, error) {
	var (
		oldState externalAccountsState
		from     connector.PSPAccount
	)
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}
	}

	if req.FromPayload == nil {
		return connector.FetchNextExternalAccountsResponse{}, errors.New("missing from payload when fetching external accounts")
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	// fetch next external accounts can be skipped in this case (eg. it is the root account)
	if from.Reference == p.client.GetRootAccountID() {
		p.logger.WithField("account_reference", from.Reference).Debugf("skipping fetch next external accounts for root account")
		return connector.FetchNextExternalAccountsResponse{}, nil
	}

	newState := oldState
	var accounts []connector.PSPAccount

	rawAccounts, timeline, hasMore, err := p.client.GetExternalAccounts(
		ctx,
		resolveAccount(from.Reference),
		oldState.Timeline,
		int64(req.PageSize),
	)
	if err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}
	newState.Timeline = timeline

	for _, acc := range rawAccounts {
		raw, err := json.Marshal(acc)
		if err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}

		metadata := make(map[string]string)
		for k, v := range acc.Metadata {
			metadata[k] = v
		}

		defaultAsset := currency.FormatAsset(supportedCurrenciesWithDecimal, string(acc.Currency))

		if acc.Account == nil {
			return connector.FetchNextExternalAccountsResponse{}, fmt.Errorf("internal account %q is missing from response for %q", from.Reference, acc.ID)
		}

		// Use import time so bank accounts don't all have a creation time of 1970-01-01T00:00:00Z
		accountCreated := time.Now().UTC()
		if acc.Account.Created > 0 {
			accountCreated = time.Unix(acc.Account.Created, 0).UTC()
		}
		accounts = append(accounts, connector.PSPAccount{
			Reference:    acc.ID,
			CreatedAt:    accountCreated,
			DefaultAsset: &defaultAsset,
			Raw:          raw,
			Metadata:     metadata,
		})
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
