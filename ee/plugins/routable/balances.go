package routable

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/routable/mappers"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextBalances refetches the account by ID so the engine sees a
// fresh available_amount even when FromPayload is stale from a previous
// accounts cycle. Routable has no streaming balance endpoint.
func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("missing from payload when fetching balances")
	}
	var from models.PSPAccount
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("decoding from payload: %w", err)
	}

	account, err := p.client.GetAccount(ctx, from.Reference)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("getting account %s: %w", from.Reference, err)
	}

	balance, err := mappers.AccountToBalance(*account, time.Now().UTC())
	if err != nil {
		p.logger.Infof("skipping balance for account %s: %v", from.Reference, err)
		return models.FetchNextBalancesResponse{HasMore: false}, nil
	}

	return models.FetchNextBalancesResponse{Balances: []models.PSPBalance{balance}, HasMore: false}, nil
}
