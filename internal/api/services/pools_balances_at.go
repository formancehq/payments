package services

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
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

	if pool.Type == models.POOL_TYPE_DYNAMIC {
		// populate the pool accounts from the query
		pool.PoolAccounts, err = s.populatePoolAccounts(ctx, pool)
		if err != nil {
			return nil, newStorageError(err, "cannot populate pool accounts")
		}
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

func (s *Service) populatePoolAccounts(ctx context.Context, pool *models.Pool) ([]models.AccountID, error) {
	queryJSON, err := json.Marshal(pool.Query)
	if err != nil {
		return nil, newStorageError(err, "cannot marshal pool query")
	}

	qb, err := query.ParseJSON(string(queryJSON))
	if err != nil {
		return nil, newStorageError(err, "cannot parse pool query")
	}

	q := storage.NewListAccountsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.AccountQuery{}).
			WithPageSize(100).
			WithQueryBuilder(qb),
	)

	res := make([]models.AccountID, 0)
	for {
		cursor, err := s.storage.AccountsList(ctx, q)
		if err != nil {
			return nil, newStorageError(err, "cannot list accounts")
		}

		for _, account := range cursor.Data {
			res = append(res, account.ID)
		}

		if !cursor.HasMore {
			break
		}

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		if err != nil {
			return nil, newStorageError(err, "cannot unmarshal cursor")
		}
	}

	return res, nil
}
