package storage

import "github.com/uptrace/bun"

type Paginator struct {
	offset uint64
	limit  uint64
}

func Paginate(offset, limit uint64) Paginator {
	return Paginator{offset, limit}
}

func (p Paginator) apply(query *bun.SelectQuery) *bun.SelectQuery {
	return query.Offset(int(p.offset)).Limit(int(p.limit))
}
