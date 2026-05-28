package fireblocks

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

// aggregatedBalance accumulates same-canonical-asset VaultAssets within a
// single vault. Multiple Fireblocks legacyIds (e.g. USDT_ERC20 + USDT_TRX)
// collapse onto one canonical asset (USDT/6); we sum their `Available` and
// keep the source legacyIds for the aggregation log line.
type aggregatedBalance struct {
	info      assetInfo
	amount    *big.Int
	legacyIDs []string
}

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, errorsutils.NewWrappedError(
			fmt.Errorf("from payload is required"),
			models.ErrInvalidRequest,
		)
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	vaultAccount, err := p.client.GetVaultAccount(ctx, from.Reference)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to get vault account %s: %w", from.Reference, err)
	}

	now := time.Now()
	agg := map[string]*aggregatedBalance{}
	for _, a := range vaultAccount.Assets {
		info, ok := p.lookupAsset(a.ID)
		if !ok {
			p.logger.Infof("skipping balance: unknown asset %q on account %s", a.ID, from.Reference)
			continue
		}

		amount, err := currency.GetAmountWithPrecisionFromString(a.Available, info.Precision)
		if err != nil {
			p.logger.Infof("skipping balance: asset %q on account %s, unparseable available %q",
				a.ID, from.Reference, a.Available)
			continue
		}

		entry, exists := agg[info.Asset]
		if !exists {
			entry = &aggregatedBalance{info: info, amount: new(big.Int)}
			agg[info.Asset] = entry
		}
		entry.amount.Add(entry.amount, amount)
		entry.legacyIDs = appendUnique(entry.legacyIDs, info.LegacyID)
	}

	balances := make([]models.PSPBalance, 0, len(agg))
	for _, entry := range agg {
		if len(entry.legacyIDs) > 1 {
			p.logger.Infof("aggregated %d fireblocks legacyIds [%s] into %s on account %s",
				len(entry.legacyIDs), strings.Join(entry.legacyIDs, ","),
				entry.info.Asset, from.Reference)
		}
		balance := models.PSPBalance{
			AccountReference: from.Reference,
			CreatedAt:        now,
			Amount:           entry.amount,
			Asset:            entry.info.Asset,
		}
		if err := balance.Validate(); err != nil {
			p.logger.Infof("dropping invalid balance for account %s asset %s: %s",
				from.Reference, entry.info.Asset, err)
			continue
		}
		balances = append(balances, balance)
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}

func appendUnique(s []string, v string) []string {
	if v == "" || slices.Contains(s, v) {
		return s
	}
	return append(s, v)
}
