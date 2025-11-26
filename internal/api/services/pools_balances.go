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

	balances, err := s.storage.BalancesGetFromAccountIDs(ctx, pool.PoolAccounts, nil)
	if err != nil {
		return nil, newStorageError(err, "cannot get latest balances")
	}

	return balances, nil
}
