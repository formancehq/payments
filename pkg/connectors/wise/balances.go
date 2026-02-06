package wise

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req connector.FetchNextBalancesRequest) (connector.FetchNextBalancesResponse, error) {
	var from connector.PSPAccount
	if req.FromPayload == nil {
		return connector.FetchNextBalancesResponse{}, connector.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	balanceID, err := strconv.ParseUint(from.Reference, 10, 64)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	pID, ok := from.Metadata[metadataProfileIDKey]
	if !ok {
		return connector.FetchNextBalancesResponse{}, errors.New("missing profile ID in from payload when fetching balances")
	}

	profileID, err := strconv.ParseUint(pID, 10, 64)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	balance, err := p.client.GetBalance(ctx, profileID, balanceID)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	precision, ok := supportedCurrenciesWithDecimal[balance.Amount.Currency]
	if !ok {
		return connector.FetchNextBalancesResponse{}, errors.New("unsupported currency")
	}

	amount, err := currency.GetAmountWithPrecisionFromString(balance.Amount.Value.String(), precision)
	if err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}

	return connector.FetchNextBalancesResponse{
		Balances: []connector.PSPBalance{
			{
				AccountReference: from.Reference,
				CreatedAt:        balance.ModificationTime,
				Amount:           amount,
				Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, balance.Amount.Currency),
			},
		},
		HasMore: false,
	}, nil
}
