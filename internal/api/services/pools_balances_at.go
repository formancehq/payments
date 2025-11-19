package services

import (
	"context"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
)

func (s *Service) PoolsBalancesAt(
	ctx context.Context,
	poolID uuid.UUID,
	at time.Time,
) ([]models.AggregatedBalance, error) {
	pool, err := s.storage.PoolsGet(ctx, poolID)
	if err != nil {
		return nil, newStorageError(err, "cannot get pool")
	}
	res := make(map[string]*aggregatedBalance)
	for i := range pool.PoolAccounts {
		balances, err := s.storage.BalancesGetAt(ctx, pool.PoolAccounts[i], at)
		if err != nil {
			return nil, newStorageError(err, "cannot get balances")
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
