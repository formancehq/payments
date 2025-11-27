package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
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

	balances, err := s.storage.BalancesGetFromAccountIDs(ctx, pool.PoolAccounts, pointer.For(at))
	if err != nil {
		return nil, newStorageError(err, "cannot get balances")
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
