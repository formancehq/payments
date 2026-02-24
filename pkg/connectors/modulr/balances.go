package modulr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req connector.FetchNextBalancesRequest) (connector.FetchNextBalancesResponse, error) {
	var from connector.PSPAccount
	if req.FromPayload == nil {
		return connector.FetchNextBalancesResponse{}, errors.New("missing from payload when fetching balances")
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	account, err := p.client.GetAccount(ctx, from.Reference)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	precision := supportedCurrenciesWithDecimal[account.Currency]

	amount, err := currency.GetAmountWithPrecisionFromString(account.Balance, precision)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse amount %s: %w", account.Balance, err)
	}

	balance := connector.PSPBalance{
		AccountReference: from.Reference,
		CreatedAt:        time.Now().UTC(),
		Amount:           amount,
		Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, account.Currency),
	}

	return connector.FetchNextBalancesResponse{
		Balances: []connector.PSPBalance{balance},
		HasMore:  false,
	}, nil
}
