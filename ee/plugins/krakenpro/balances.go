package krakenpro

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/krakenpro/mappers"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextBalances re-reads BalanceEx (cheap single-shot call) and
// emits one PSPBalance per raw Kraken variant, each keyed to its own
// per-class account (spot, staked, earn, …). No aggregation: distinct
// account references mean the engine never sees a duplicate
// (account, asset) tuple, so each variant reports its real balance.
// See MAPPINGS §6.
func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	currencies, _, err := p.ensureAssets(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	entries, err := p.client.GetBalanceEx(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to fetch balance ex: %w", err)
	}

	now := time.Now().UTC()
	balances := make([]models.PSPBalance, 0, len(entries))
	for rawCode, entry := range entries {
		bal, mapErr := mappers.RawBalanceToPSPBalance(currencies, rawCode, entry, now)
		if mapErr != nil {
			p.logger.WithField("rawCode", rawCode).Errorf("map balance: %v", mapErr)
			continue
		}
		if bal == nil {
			continue
		}
		balances = append(balances, *bal)
	}

	p.logger.WithField("emitted", len(balances)).WithField("rawEntries", len(entries)).
		Infof("krakenpro fetch_balances cycle done")
	return models.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}
