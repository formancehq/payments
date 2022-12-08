package storage

import "github.com/uptrace/bun"

type Paginator struct {
	offset uint
	limit  uint
}

func Paginate(offset, limit uint) Paginator {
	return Paginator{offset, limit}
}

func (p Paginator) apply(query *bun.SelectQuery) *bun.SelectQuery {
	return query.Offset(int(p.offset)).Limit(int(p.limit))
}
