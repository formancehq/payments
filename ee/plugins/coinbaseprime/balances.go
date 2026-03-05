package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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

	response, err := p.client.GetBalanceForWallet(ctx, from.Reference)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	bal := response.Balance
	symbol := strings.ToUpper(strings.TrimSpace(bal.Symbol))

	precision, ok := p.currencies[symbol]
	if !ok {
		return models.FetchNextBalancesResponse{
			Balances: nil,
			HasMore:  false,
		}, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(bal.Amount, precision)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse balance for %s: %w", symbol, err)
	}

	now := time.Now().UTC()
	asset := currency.FormatAsset(p.currencies, symbol)

	return models.FetchNextBalancesResponse{
		Balances: []models.PSPBalance{{
			AccountReference: from.Reference,
			Asset:            asset,
			Amount:           amount,
			CreatedAt:        now,
		}},
		HasMore: false,
	}, nil
}
