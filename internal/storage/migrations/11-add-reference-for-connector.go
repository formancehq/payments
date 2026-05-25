package migrations

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/uptrace/bun"
)

func AddReferenceForConnector(ctx context.Context, db bun.IDB) error {
	_, err := db.ExecContext(ctx, `
		ALTER TABLE connectors ADD COLUMN IF NOT EXISTS reference uuid;
	`)
	if err != nil {
		return err
	}

	q := paginate.OffsetPaginatedQuery[paginate.PaginatedQueryOptions[ConnectorQuery]]{
		Order:    paginate.OrderAsc,
		PageSize: 100,
		Options: paginate.PaginatedQueryOptions[ConnectorQuery]{
			PageSize: 100,
			Options:  ConnectorQuery{},
		},
	}

	for {
		cursor, err := paginateWithOffset[paginate.PaginatedQueryOptions[ConnectorQuery], v3Connector](
			ctx,
			db,
			(*paginate.OffsetPaginatedQuery[paginate.PaginatedQueryOptions[ConnectorQuery]])(&q),
			func(query *bun.SelectQuery) *bun.SelectQuery {
				return query.
					Column("id").
					Order("created_at ASC", "sort_id ASC")
			},
		)
		if err != nil {
			return err
		}

		for _, connector := range cursor.Data {
			_, err = db.NewUpdate().
				Model((*v3Connector)(nil)).
				Set("reference = ?", connector.ID.Reference).
				Where("id = ?", connector.ID.String()).
				Exec(ctx)
			if err != nil {
				return err
			}
		}

		if !cursor.HasMore {
			break
		}

		err = paginate.UnmarshalCursor(cursor.Next, &q)
		if err != nil {
			return err
		}
	}

	_, err = db.ExecContext(ctx, `
		ALTER TABLE connectors ALTER COLUMN reference SET NOT NULL;
	`)
	if err != nil {
		return err
	}

	return nil
}
