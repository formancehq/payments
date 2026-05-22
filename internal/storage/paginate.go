package storage

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/uptrace/bun"
)

func paginateWithOffset[FILTERS any, RETURN any](s *store, ctx context.Context,
	q *paginate.OffsetPaginatedQuery[FILTERS], builders ...func(query *bun.SelectQuery) *bun.SelectQuery) (*paginate.Cursor[RETURN], error) {
	query := s.db.NewSelect()
	return paginate.UsingOffset[FILTERS, RETURN](ctx, query, *q, builders...)
}
