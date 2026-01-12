package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	internalTime "github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
)

type order struct {
	bun.BaseModel `bun:"table:orders"`

	// Mandatory fields
	ID                  models.OrderID       `bun:"id,pk,type:character varying,notnull"`
	ConnectorID         models.ConnectorID   `bun:"connector_id,type:character varying,notnull"`
	Reference           string               `bun:"reference,type:text,notnull"`
	CreatedAt           internalTime.Time    `bun:"created_at,type:timestamp without time zone,notnull"`
	UpdatedAt           internalTime.Time    `bun:"updated_at,type:timestamp without time zone,notnull"`
	Direction           models.OrderDirection `bun:"direction,type:text,notnull"`
	SourceAsset         string               `bun:"source_asset,type:text,notnull"`
	TargetAsset         string               `bun:"target_asset,type:text,notnull"`
	Type                models.OrderType     `bun:"type,type:text,notnull"`
	BaseQuantityOrdered *big.Int             `bun:"base_quantity_ordered,type:numeric,notnull"`
	TimeInForce         models.TimeInForce   `bun:"time_in_force,type:text,notnull"`

	// Scan only fields - status is derived from adjustments
	Status models.OrderStatus `bun:"status,type:text,notnull,scanonly"`

	// Optional fields
	BaseQuantityFilled *big.Int          `bun:"base_quantity_filled,type:numeric,nullzero"`
	LimitPrice         *big.Int          `bun:"limit_price,type:numeric,nullzero"`
	ExpiresAt          *internalTime.Time `bun:"expires_at,type:timestamp without time zone,nullzero"`
	Fee                *big.Int          `bun:"fee,type:numeric,nullzero"`
	FeeAsset           *string           `bun:"fee_asset,type:text,nullzero"`
	AverageFillPrice   *big.Int          `bun:"average_fill_price,type:numeric,nullzero"`

	// Optional fields with default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

type orderAdjustment struct {
	bun.BaseModel `bun:"table:order_adjustments"`

	// Mandatory fields
	ID        models.OrderAdjustmentID `bun:"id,pk,type:character varying,notnull"`
	OrderID   models.OrderID           `bun:"order_id,type:character varying,notnull"`
	Reference string                   `bun:"reference,type:text,notnull"`
	CreatedAt internalTime.Time        `bun:"created_at,type:timestamp without time zone,notnull"`
	Status    models.OrderStatus       `bun:"status,type:text,notnull"`
	Raw       json.RawMessage          `bun:"raw,type:json,notnull"`

	// Optional fields
	BaseQuantityFilled *big.Int `bun:"base_quantity_filled,type:numeric,nullzero"`
	Fee                *big.Int `bun:"fee,type:numeric,nullzero"`
	FeeAsset           *string  `bun:"fee_asset,type:text,nullzero"`

	// Optional fields with default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

func (s *store) OrdersUpsert(ctx context.Context, orders []models.Order) error {
	ordersToInsert := make([]order, 0, len(orders))
	adjustmentsToInsert := make([]orderAdjustment, 0)

	for _, o := range orders {
		ordersToInsert = append(ordersToInsert, fromOrderModels(o))

		for _, a := range o.Adjustments {
			adjustmentsToInsert = append(adjustmentsToInsert, fromOrderAdjustmentModels(a))
		}
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return e("failed to create transaction", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	if len(ordersToInsert) > 0 {
		_, err = tx.NewInsert().
			Model(&ordersToInsert).
			On("CONFLICT (id) DO UPDATE").
			Set("updated_at = EXCLUDED.updated_at").
			Set("status = EXCLUDED.status").
			Set("base_quantity_filled = EXCLUDED.base_quantity_filled").
			Set("fee = EXCLUDED.fee").
			Set("fee_asset = EXCLUDED.fee_asset").
			Set("average_fill_price = EXCLUDED.average_fill_price").
			Set("metadata = order.metadata || EXCLUDED.metadata").
			Exec(ctx)
		if err != nil {
			return e("failed to insert orders", err)
		}
	}

	if len(adjustmentsToInsert) > 0 {
		_, err = tx.NewInsert().
			Model(&adjustmentsToInsert).
			On("CONFLICT (id) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return e("failed to insert order adjustments", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return e("failed to commit transaction", err)
	}

	return nil
}

func (s *store) OrdersGet(ctx context.Context, id models.OrderID) (*models.Order, error) {
	var o order
	err := s.db.NewSelect().
		Model(&o).
		Column("order.*", "oad.status").
		Join(`join lateral (
			select status
			from order_adjustments oad
			where order_id = "order".id
			order by created_at desc, sort_id desc
			limit 1
		) oad on true`).
		Where("order.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("failed to get order", err)
	}

	adjustments, err := s.getOrderAdjustments(ctx, id)
	if err != nil {
		return nil, err
	}

	res := toOrderModels(o)
	res.Adjustments = adjustments

	return &res, nil
}

func (s *store) getOrderAdjustments(ctx context.Context, orderID models.OrderID) ([]models.OrderAdjustment, error) {
	var adjustments []orderAdjustment
	err := s.db.NewSelect().
		Model(&adjustments).
		Where("order_id = ?", orderID).
		Order("created_at ASC", "sort_id ASC").
		Scan(ctx)
	if err != nil {
		return nil, e("failed to get order adjustments", err)
	}

	res := make([]models.OrderAdjustment, 0, len(adjustments))
	for _, a := range adjustments {
		res = append(res, toOrderAdjustmentModels(a))
	}

	return res, nil
}

func (s *store) OrdersDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*order)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)

	return e("failed to delete orders", err)
}

func (s *store) OrdersDelete(ctx context.Context, id models.OrderID) error {
	_, err := s.db.NewDelete().
		Model((*order)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return e("failed to delete order", err)
}

func (s *store) OrdersUpdateStatus(ctx context.Context, id models.OrderID, status models.OrderStatus) error {
	now := time.Now().UTC()
	adj := orderAdjustment{
		ID: models.OrderAdjustmentID{
			OrderID:   id,
			Reference: fmt.Sprintf("status-update-%d", now.UnixNano()),
			CreatedAt: now,
			Status:    status,
		},
		OrderID:   id,
		Reference: fmt.Sprintf("status-update-%d", now.UnixNano()),
		CreatedAt: internalTime.New(now),
		Status:    status,
		Raw:       json.RawMessage(`{"source": "formance"}`),
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return e("failed to create transaction", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	// Update order status
	_, err = tx.NewUpdate().
		Model((*order)(nil)).
		Set("status = ?", status).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return e("failed to update order status", err)
	}

	// Insert adjustment
	_, err = tx.NewInsert().
		Model(&adj).
		Exec(ctx)
	if err != nil {
		return e("failed to insert order adjustment", err)
	}

	err = tx.Commit()
	if err != nil {
		return e("failed to commit transaction", err)
	}

	return nil
}

type OrderQuery struct{}

type ListOrdersQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[OrderQuery]]

func NewListOrdersQuery(opts bunpaginate.PaginatedQueryOptions[OrderQuery]) ListOrdersQuery {
	return ListOrdersQuery{
		PageSize: opts.PageSize,
		Order:    bunpaginate.OrderAsc,
		Options:  opts,
	}
}

func (s *store) ordersQueryContext(qb query.Builder) (string, []any, error) {
	where, args, err := qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch {
		case key == "reference",
			key == "id",
			key == "connector_id",
			key == "direction",
			key == "source_asset",
			key == "target_asset",
			key == "type",
			key == "status",
			key == "time_in_force":
			if operator != "$match" {
				return "", nil, e(fmt.Sprintf("'%s' column can only be used with $match", key), ErrValidation)
			}
			return fmt.Sprintf("%s = ?", key), []any{value}, nil

		case key == "base_quantity_ordered",
			key == "base_quantity_filled",
			key == "limit_price",
			key == "fee":
			return fmt.Sprintf("%s %s ?", key, query.DefaultComparisonOperatorsMapping[operator]), []any{value}, nil
		case metadataRegex.Match([]byte(key)):
			if operator != "$match" {
				return "", nil, e("'metadata' column can only be used with $match", ErrValidation)
			}
			match := metadataRegex.FindAllStringSubmatch(key, 3)

			key := "metadata"
			return key + " @> ?", []any{map[string]any{
				match[0][1]: value,
			}}, nil
		default:
			return "", nil, fmt.Errorf("unknown key '%s' when building query: %w", key, ErrValidation)
		}
	}))

	return where, args, err
}

func (s *store) OrdersList(ctx context.Context, q ListOrdersQuery) (*bunpaginate.Cursor[models.Order], error) {
	var (
		where string
		args  []any
		err   error
	)
	if q.Options.QueryBuilder != nil {
		where, args, err = s.ordersQueryContext(q.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[OrderQuery], order](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[OrderQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			if where != "" {
				query = query.Where(where, args...)
			}

			query.Column("order.*", "oad.status").
				Join(`join lateral (
				select status
				from order_adjustments oad
				where order_id = "order".id
				order by created_at desc, sort_id desc
				limit 1
			) oad on true`)

			query = query.Order("created_at DESC", "sort_id DESC")

			return query
		},
	)
	if err != nil {
		return nil, err
	}

	orders := make([]models.Order, 0, len(cursor.Data))
	for _, o := range cursor.Data {
		orders = append(orders, toOrderModels(o))
	}

	return &bunpaginate.Cursor[models.Order]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     orders,
	}, nil
}

func fromOrderModels(from models.Order) order {
	o := order{
		ID:                  from.ID,
		ConnectorID:         from.ConnectorID,
		Reference:           from.Reference,
		CreatedAt:           internalTime.New(from.CreatedAt),
		UpdatedAt:           internalTime.New(from.UpdatedAt),
		Direction:           from.Direction,
		SourceAsset:         from.SourceAsset,
		TargetAsset:         from.TargetAsset,
		Type:                from.Type,
		Status:              from.Status,
		BaseQuantityOrdered: from.BaseQuantityOrdered,
		BaseQuantityFilled:  from.BaseQuantityFilled,
		LimitPrice:          from.LimitPrice,
		Fee:                 from.Fee,
		FeeAsset:            from.FeeAsset,
		AverageFillPrice:    from.AverageFillPrice,
		TimeInForce:         from.TimeInForce,
		Metadata:            from.Metadata,
	}

	if from.ExpiresAt != nil {
		t := internalTime.New(*from.ExpiresAt)
		o.ExpiresAt = &t
	}

	return o
}

func toOrderModels(from order) models.Order {
	o := models.Order{
		ID:                  from.ID,
		ConnectorID:         from.ConnectorID,
		Reference:           from.Reference,
		CreatedAt:           from.CreatedAt.Time,
		UpdatedAt:           from.UpdatedAt.Time,
		Direction:           from.Direction,
		SourceAsset:         from.SourceAsset,
		TargetAsset:         from.TargetAsset,
		Type:                from.Type,
		Status:              from.Status,
		BaseQuantityOrdered: from.BaseQuantityOrdered,
		BaseQuantityFilled:  from.BaseQuantityFilled,
		LimitPrice:          from.LimitPrice,
		Fee:                 from.Fee,
		FeeAsset:            from.FeeAsset,
		AverageFillPrice:    from.AverageFillPrice,
		TimeInForce:         from.TimeInForce,
		Metadata:            from.Metadata,
	}

	if from.ExpiresAt != nil {
		o.ExpiresAt = &from.ExpiresAt.Time
	}

	return o
}

func fromOrderAdjustmentModels(from models.OrderAdjustment) orderAdjustment {
	return orderAdjustment{
		ID:                 from.ID,
		OrderID:            from.ID.OrderID,
		Reference:          from.Reference,
		CreatedAt:          internalTime.New(from.CreatedAt),
		Status:             from.Status,
		BaseQuantityFilled: from.BaseQuantityFilled,
		Fee:                from.Fee,
		FeeAsset:           from.FeeAsset,
		Metadata:           from.Metadata,
		Raw:                from.Raw,
	}
}

func toOrderAdjustmentModels(from orderAdjustment) models.OrderAdjustment {
	return models.OrderAdjustment{
		ID:                 from.ID,
		Reference:          from.Reference,
		CreatedAt:          from.CreatedAt.Time,
		Status:             from.Status,
		BaseQuantityFilled: from.BaseQuantityFilled,
		Fee:                from.Fee,
		FeeAsset:           from.FeeAsset,
		Metadata:           from.Metadata,
		Raw:                from.Raw,
	}
}
