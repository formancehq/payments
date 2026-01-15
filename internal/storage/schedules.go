package storage

import (
	"context"
	"fmt"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
)

type schedule struct {
	bun.BaseModel `bun:"table:schedules"`

	// Mandatory fields
	ID          string             `bun:"id,pk,type:text,notnull"`
	ConnectorID models.ConnectorID `bun:"connector_id,pk,type:character varying,notnull"`
	CreatedAt   time.Time          `bun:"created_at,type:timestamp without time zone,notnull"`
}

func (s *store) SchedulesUpsert(ctx context.Context, schedule models.Schedule) error {
	toInsert := fromScheduleModel(schedule)

	_, err := s.db.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id, connector_id) DO NOTHING").
		Exec(ctx)

	return e("failed to insert schedule", err)
}

func (s *store) SchedulesDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*schedule)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)

	return e("failed to delete schedule", err)
}

// SchedulesDeleteFromConnectorIDBatch deletes a batch of schedules for a given connector ID
// and returns the number of rows affected
func (s *store) SchedulesDeleteFromConnectorIDBatch(ctx context.Context, connectorID models.ConnectorID, batchSize int) (int, error) {
	result, err := s.db.NewDelete().
		Model((*schedule)(nil)).
		Where("(id, connector_id) IN (SELECT id, connector_id FROM schedules WHERE connector_id = ? LIMIT ?)", connectorID, batchSize).
		Exec(ctx)
	if err != nil {
		return 0, e("failed to delete schedules batch", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, e("failed to get rows affected", err)
	}

	return int(rowsAffected), nil
}

func (s *store) SchedulesDelete(ctx context.Context, id string) error {
	_, err := s.db.NewDelete().
		Model((*schedule)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return e("failed to delete schedule", err)
}

func (s *store) SchedulesGet(ctx context.Context, id string, connectorID models.ConnectorID) (*models.Schedule, error) {
	var schedule schedule
	err := s.db.NewSelect().
		Model(&schedule).
		Where("id = ? AND connector_id = ?", id, connectorID).
		Scan(ctx)

	if err != nil {
		return nil, e("failed to fetch schedule", err)
	}

	return pointer.For(toScheduleModel(schedule)), nil
}

type ScheduleQuery struct{}

type ListSchedulesQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[ScheduleQuery]]

func NewListSchedulesQuery(opts bunpaginate.PaginatedQueryOptions[ScheduleQuery]) ListSchedulesQuery {
	return ListSchedulesQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) schedulesQueryContext(qb query.Builder) (string, []any, error) {
	return qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch key {
		case "id", "connector_id":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}
			return fmt.Sprintf("%s = ?", key), []any{value}, nil
		default:
			return "", nil, fmt.Errorf("unknown key '%s' when building query: %w", key, ErrValidation)
		}
	}))
}

func (s *store) SchedulesList(ctx context.Context, q ListSchedulesQuery) (*bunpaginate.Cursor[models.Schedule], error) {
	var (
		where string
		args  []any
		err   error
	)
	if q.Options.QueryBuilder != nil {
		where, args, err = s.schedulesQueryContext(q.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[ScheduleQuery], schedule](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[ScheduleQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			if where != "" {
				query = query.Where(where, args...)
			}

			query = query.Order("created_at DESC", "sort_id DESC")

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch schedules", err)
	}

	schedules := make([]models.Schedule, 0, len(cursor.Data))
	for _, s := range cursor.Data {
		schedules = append(schedules, toScheduleModel(s))
	}

	return &bunpaginate.Cursor[models.Schedule]{
		PageSize: cursor.PageSize,
		Data:     schedules,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
	}, nil
}

func fromScheduleModel(s models.Schedule) schedule {
	return schedule{
		ID:          s.ID,
		ConnectorID: s.ConnectorID,
		CreatedAt:   time.New(s.CreatedAt),
	}
}

func toScheduleModel(s schedule) models.Schedule {
	return models.Schedule{
		ID:          s.ID,
		ConnectorID: s.ConnectorID,
		CreatedAt:   s.CreatedAt.Time,
	}
}
