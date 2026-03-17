package krakenpro

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
)


func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("missing from payload when fetching balances")
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	response, err := p.client.GetBalance(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	now := time.Now().UTC()
	balances := make([]models.PSPBalance, 0, len(response.Result))

	// Sort keys for deterministic output
	assets := make([]string, 0, len(response.Result))
	for asset := range response.Result {
		assets = append(assets, asset)
	}
	sort.Strings(assets)

	for _, rawAsset := range assets {
		amountStr := response.Result[rawAsset]

		// Skip zero balances
		amt, ok := new(big.Float).SetString(amountStr)
		if !ok {
			p.logger.Infof("skipping asset %q: invalid amount %q", rawAsset, amountStr)
			continue
		}
		if amt.Sign() == 0 {
			continue
		}

		normalized := normalizeAssetCode(rawAsset)
		if normalized == "" {
			continue
		}

		precision := p.getPrecision(normalized)

		amount, err := currency.GetAmountWithPrecisionFromString(truncateToPrecision(amountStr, precision), precision)
		if err != nil {
			p.logger.Infof("skipping balance for %s: %v", normalized, err)
			continue
		}

		// Use the normalized code to format the asset, falling back to the
		// raw precision when the code is not in the currency library's map.
		asset := currency.FormatAsset(p.formattedCurrMap, normalized)

		balances = append(balances, models.PSPBalance{
			AccountReference: from.Reference,
			Asset:            asset,
			Amount:           amount,
			CreatedAt:        now,
		})
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}

