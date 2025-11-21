package services

import (
	"context"
	"math/big"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

type aggregatedBalance struct {
	amount          *big.Int
	relatedAccounts []models.AccountID
}

func (s *Service) PoolsBalances(
	ctx context.Context,
	poolID uuid.UUID,
) ([]models.AggregatedBalance, error) {
	pool, err := s.storage.PoolsGet(ctx, poolID)
	if err != nil {
		return nil, newStorageError(err, "cannot get pool")
	}

	if pool.Type == models.POOL_TYPE_DYNAMIC {
		// populate the pool accounts from the query
		pool.PoolAccounts, err = s.populatePoolAccounts(ctx, pool)
		if err != nil {
			return nil, newStorageError(err, "cannot populate pool accounts")
		}
	}

	res := make(map[string]*aggregatedBalance)
	for i := range pool.PoolAccounts {
		balances, err := s.storage.BalancesGetLatest(ctx, pool.PoolAccounts[i])
		if err != nil {
			return nil, newStorageError(err, "cannot get latest balances")
		}

		for _, balance := range balances {
			v, ok := res[balance.Asset]
			if !ok {
				v = &aggregatedBalance{
					amount:          big.NewInt(0),
					relatedAccounts: []models.AccountID{},
				}
			}

			v.amount = v.amount.Add(v.amount, balance.Balance)
			v.relatedAccounts = append(v.relatedAccounts, balance.AccountID)
			res[balance.Asset] = v
		}
	}

	balances := make([]models.AggregatedBalance, 0, len(res))
	for asset, balance := range res {
		balances = append(balances, models.AggregatedBalance{
			Asset:           asset,
			Amount:          balance.amount,
			RelatedAccounts: balance.relatedAccounts,
		})
	}

	return balances, nil
}
