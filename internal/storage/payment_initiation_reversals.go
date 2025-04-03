package storage

import (
	"context"
	"fmt"
	"math/big"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/go-libs/v2/query"
	"github.com/formancehq/go-libs/v2/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

type paymentInitiationReversal struct {
	bun.BaseModel `bun:"payment_initiation_reversals"`

	// Mandatory fields
	ID                  models.PaymentInitiationReversalID `bun:"id,pk,type:character varying,notnull"`
	ConnectorID         models.ConnectorID                 `bun:"connector_id,type:character varying,notnull"`
	PaymentInitiationID models.PaymentInitiationID         `bun:"payment_initiation_id,type:character varying,notnull"`
	Reference           string                             `bun:"reference,type:text,notnull"`
	CreatedAt           time.Time                          `bun:"created_at,type:timestamp without time zone,notnull"`
	Description         string                             `bun:"description,type:text,notnull"`
	Amount              *big.Int                           `bun:"amount,type:numeric,notnull"`
	Asset               string                             `bun:"asset,type:text,notnull"`

	// Optional fields with default
	// c.f. https://bun.uptrace.dev/guide/models.html#default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

type paymentInitiationReversalAdjustment struct {
	bun.BaseModel `bun:"payment_initiation_reversal_adjustments"`

	// Mandatory fields
	ID                          models.PaymentInitiationReversalAdjustmentID `bun:"id,pk,type:character varying,notnull"`
	PaymentInitiationReversalID models.PaymentInitiationReversalID           `bun:"payment_initiation_reversal_id,type:character varying,notnull"`
	CreatedAt                   time.Time                                    `bun:"created_at,type:timestamp without time zone,notnull"`
	Status                      models.PaymentInitiationReversalAdjustmentStatus

	// Optional fields
	Error *string `bun:"error,type:text"`

	// Optional fields with default
	// c.f. https://bun.uptrace.dev/guide/models.html#default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

func (s *store) PaymentInitiationReversalsUpsert(
	ctx context.Context,
	pir models.PaymentInitiationReversal,
	reversalAdjustments []models.PaymentInitiationReversalAdjustment,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e("upsert payment initiation reversal", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	toInsert := fromPaymentInitiationReversalModels(pir)
	reversalAdjustementsToInsert := make([]paymentInitiationReversalAdjustment, 0, len(reversalAdjustments))
	for _, adj := range reversalAdjustments {
		reversalAdjustementsToInsert = append(reversalAdjustementsToInsert, fromPaymentInitiationReversalAdjustmentModels(adj))
	}

	_, err = tx.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return e("upsert payment initiation reversal", err)
	}

	if len(reversalAdjustementsToInsert) > 0 {
		_, err = tx.NewInsert().
			Model(&reversalAdjustementsToInsert).
			On("CONFLICT (id) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return e("upsert payment initiation reversal adjustments", err)
		}
	}

	return e("failed to commit transaction", tx.Commit())
}

func (s *store) PaymentInitiationReversalsGet(ctx context.Context, id models.PaymentInitiationReversalID) (*models.PaymentInitiationReversal, error) {
	var pir paymentInitiationReversal
	err := s.db.NewSelect().
		Model(&pir).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("get payment initiation reversal", err)
	}

	res := toPaymentInitiationReversalModels(pir)
	return &res, nil
}

func (s *store) PaymentInitiationReversalsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*paymentInitiationReversal)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)
	if err != nil {
		return e("delete payment initiation reversal", err)
	}

	return nil
}

type PaymentInitiationReversalQuery struct{}

type ListPaymentInitiationReversalsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PaymentInitiationReversalQuery]]

func NewListPaymentInitiationReversalsQuery(opts bunpaginate.PaginatedQueryOptions[PaymentInitiationReversalQuery]) ListPaymentInitiationReversalsQuery {
	return ListPaymentInitiationReversalsQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) paymentsInitiationReversalQueryContext(qb query.Builder) (string, []any, error) {
	where, args, err := qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch {
		case key == "reference",
			key == "id",
			key == "connector_id",
			key == "asset",
			key == "payment_initiation_id":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}
			return fmt.Sprintf("%s = ?", key), []any{value}, nil

		case key == "amount":
			return fmt.Sprintf("%s %s ?", key, query.DefaultComparisonOperatorsMapping[operator]), []any{value}, nil
		case metadataRegex.Match([]byte(key)):
			if operator != "$match" {
				return "", nil, errors.Wrap(ErrValidation, "'metadata' column can only be used with $match")
			}
			match := metadataRegex.FindAllStringSubmatch(key, 3)

			key := "metadata"
			return key + " @> ?", []any{map[string]any{
				match[0][1]: value,
			}}, nil
		default:
			return "", nil, errors.Wrap(ErrValidation, fmt.Sprintf("unknown key '%s' when building query", key))
		}
	}))

	return where, args, err
}

func (s *store) PaymentInitiationReversalsList(ctx context.Context, q ListPaymentInitiationReversalsQuery) (*bunpaginate.Cursor[models.PaymentInitiationReversal], error) {
	var (
		where string
		args  []any
		err   error
	)
	if q.Options.QueryBuilder != nil {
		where, args, err = s.paymentsInitiationReversalQueryContext(q.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PaymentInitiationReversalQuery], paymentInitiationReversal](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PaymentInitiationReversalQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			if where != "" {
				query = query.Where(where, args...)
			}

			// TODO(polo): sorter ?
			query = query.Order("created_at DESC", "sort_id DESC")

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch payment initiation reversals", err)
	}

	pis := make([]models.PaymentInitiationReversal, 0, len(cursor.Data))
	for _, pi := range cursor.Data {
		pis = append(pis, toPaymentInitiationReversalModels(pi))
	}

	return &bunpaginate.Cursor[models.PaymentInitiationReversal]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     pis,
	}, nil
}

func (s *store) PaymentInitiationReversalAdjustmentsUpsert(ctx context.Context, adj models.PaymentInitiationReversalAdjustment) error {
	toInsert := fromPaymentInitiationReversalAdjustmentModels(adj)

	_, err := s.db.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return e("upsert payment initiation reversal adjustment", err)
	}

	return nil
}

func (s *store) PaymentInitiationReversalAdjustmentsGet(ctx context.Context, id models.PaymentInitiationReversalAdjustmentID) (*models.PaymentInitiationReversalAdjustment, error) {
	var adj paymentInitiationReversalAdjustment
	err := s.db.NewSelect().
		Model(&adj).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("failed to get payment initiation reversal adjustment", err)
	}

	res := toPaymentInitiationReversalAdjustmentModels(adj)
	return &res, nil
}

type PaymentInitiationReversalAdjustmentsQuery struct{}

type ListPaymentInitiationReversalAdjustmentsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PaymentInitiationReversalAdjustmentsQuery]]

func NewListPaymentInitiationReversalAdjustmentsQuery(opts bunpaginate.PaginatedQueryOptions[PaymentInitiationReversalAdjustmentsQuery]) ListPaymentInitiationReversalAdjustmentsQuery {
	return ListPaymentInitiationReversalAdjustmentsQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) PaymentInitiationReversalAdjustmentsList(ctx context.Context, piID models.PaymentInitiationReversalID, q ListPaymentInitiationReversalAdjustmentsQuery) (*bunpaginate.Cursor[models.PaymentInitiationReversalAdjustment], error) {
	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PaymentInitiationReversalAdjustmentsQuery], paymentInitiationReversalAdjustment](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PaymentInitiationReversalAdjustmentsQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			// TODO(polo): sorter ?
			query = query.Order("created_at DESC", "sort_id DESC")
			query.Where("payment_initiation_reversal_id = ?", piID)

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch accounts", err)
	}

	pis := make([]models.PaymentInitiationReversalAdjustment, 0, len(cursor.Data))
	for _, pi := range cursor.Data {
		pis = append(pis, toPaymentInitiationReversalAdjustmentModels(pi))
	}

	return &bunpaginate.Cursor[models.PaymentInitiationReversalAdjustment]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     pis,
	}, nil
}

func fromPaymentInitiationReversalModels(from models.PaymentInitiationReversal) paymentInitiationReversal {
	return paymentInitiationReversal{
		ID:                  from.ID,
		ConnectorID:         from.ConnectorID,
		PaymentInitiationID: from.PaymentInitiationID,
		Reference:           from.Reference,
		CreatedAt:           time.New(from.CreatedAt),
		Description:         from.Description,
		Amount:              from.Amount,
		Asset:               from.Asset,
		Metadata:            from.Metadata,
	}
}

func toPaymentInitiationReversalModels(from paymentInitiationReversal) models.PaymentInitiationReversal {
	return models.PaymentInitiationReversal{
		ID:                  from.ID,
		ConnectorID:         from.ConnectorID,
		PaymentInitiationID: from.PaymentInitiationID,
		Reference:           from.Reference,
		CreatedAt:           from.CreatedAt.Time,
		Description:         from.Description,
		Amount:              from.Amount,
		Asset:               from.Asset,
		Metadata:            from.Metadata,
	}
}

func fromPaymentInitiationReversalAdjustmentModels(from models.PaymentInitiationReversalAdjustment) paymentInitiationReversalAdjustment {
	return paymentInitiationReversalAdjustment{
		ID:                          from.ID,
		PaymentInitiationReversalID: from.PaymentInitiationReversalID,
		CreatedAt:                   time.New(from.CreatedAt),
		Status:                      from.Status,
		Error: func() *string {
			if from.Error == nil {
				return nil
			}
			return pointer.For(from.Error.Error())
		}(),
		Metadata: from.Metadata,
	}
}

func toPaymentInitiationReversalAdjustmentModels(from paymentInitiationReversalAdjustment) models.PaymentInitiationReversalAdjustment {
	return models.PaymentInitiationReversalAdjustment{
		ID:                          from.ID,
		PaymentInitiationReversalID: from.PaymentInitiationReversalID,
		CreatedAt:                   from.CreatedAt.Time,
		Status:                      from.Status,
		Error: func() error {
			if from.Error == nil {
				return nil
			}

			return errors.New(*from.Error)
		}(),
		Metadata: from.Metadata,
	}
}
