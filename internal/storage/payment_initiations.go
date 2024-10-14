package storage

import (
	"context"
	"fmt"
	"math/big"
	stdtime "time"

	"github.com/formancehq/go-libs/bun/bunpaginate"
	"github.com/formancehq/go-libs/pointer"
	"github.com/formancehq/go-libs/query"
	"github.com/formancehq/go-libs/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

type paymentInitiation struct {
	bun.BaseModel `bun:"payment_initiations"`

	// Mandatory fields
	ID          models.PaymentInitiationID   `bun:"id,pk,type:character varying,notnull"`
	ConnectorID models.ConnectorID           `bun:"connector_id,type:character varying,notnull"`
	Reference   string                       `bun:"reference,type:text,notnull"`
	CreatedAt   time.Time                    `bun:"created_at,type:timestamp without time zone,notnull"`
	ScheduledAt time.Time                    `bun:"scheduled_at,type:timestamp without time zone,notnull"`
	Description string                       `bun:"description,type:text,notnull"`
	Type        models.PaymentInitiationType `bun:"type,type:text,notnull"`
	Amount      *big.Int                     `bun:"amount,type:numeric,notnull"`
	Asset       string                       `bun:"asset,type:text,notnull"`

	// Optional fields
	SourceAccountID      *models.AccountID `bun:"source_account_id,type:character varying"`
	DestinationAccountID *models.AccountID `bun:"destination_account_id,type:character varying,notnull"`

	// Optional fields with default
	// c.f. https://bun.uptrace.dev/guide/models.html#default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

type paymentInitiationRelatedPayment struct {
	bun.BaseModel `bun:"payment_initiation_related_payments"`

	// Mandatory fields
	PaymentInitiationID models.PaymentInitiationID `bun:"payment_initiation_id,pk,type:character varying,notnull"`
	PaymentID           models.PaymentID           `bun:"payment_id,pk,type:character varying,notnull"`
	CreatedAt           time.Time                  `bun:"created_at,type:timestamp without time zone,notnull"`
}

type paymentInitiationAdjustment struct {
	bun.BaseModel `bun:"payment_initiation_adjustments"`

	// Mandatory fields
	ID                  models.PaymentInitiationAdjustmentID     `bun:"id,pk,type:character varying,notnull"`
	PaymentInitiationID models.PaymentInitiationID               `bun:"payment_initiation_id,type:character varying,notnull"`
	CreatedAt           time.Time                                `bun:"created_at,type:timestamp without time zone,notnull"`
	Status              models.PaymentInitiationAdjustmentStatus `bun:"status,type:text,notnull"`

	// Optional fields
	Error *string `bun:"error,type:text"`

	// Optional fields with default
	// c.f. https://bun.uptrace.dev/guide/models.html#default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

func (s *store) PaymentInitiationsUpsert(ctx context.Context, pi models.PaymentInitiation, adjustments ...models.PaymentInitiationAdjustment) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e("update payment metadata", err)
	}
	defer tx.Rollback()

	toInsert := fromPaymentInitiationModels(pi)
	adjustmentsToInsert := make([]paymentInitiationAdjustment, 0, len(adjustments))
	for _, adj := range adjustments {
		adjustmentsToInsert = append(adjustmentsToInsert, fromPaymentInitiationAdjustmentModels(adj))
	}

	_, err = tx.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return e("failed to insert payment initiations", err)
	}

	if len(adjustmentsToInsert) > 0 {
		_, err = tx.NewInsert().
			Model(&adjustmentsToInsert).
			On("CONFLICT (id) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return e("failed to insert payment initiation adjustments", err)
		}
	}

	return e("failed to commit transaction", tx.Commit())
}

func (s *store) PaymentInitiationsUpdateMetadata(ctx context.Context, piID models.PaymentInitiationID, metadata map[string]string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e("update payment metadata", err)
	}
	defer tx.Rollback()

	var pi paymentInitiation
	err = tx.NewSelect().
		Model(&pi).
		Column("id", "metadata").
		Where("id = ?", piID).
		Scan(ctx)
	if err != nil {
		return e("update payment initiation metadata", err)
	}

	if pi.Metadata == nil {
		pi.Metadata = make(map[string]string)
	}

	for k, v := range metadata {
		pi.Metadata[k] = v
	}

	_, err = tx.NewUpdate().
		Model(&pi).
		Column("metadata").
		Where("id = ?", piID).
		Exec(ctx)
	if err != nil {
		return e("update payment initiation metadata", err)
	}

	return e("failed to commit transaction", tx.Commit())
}

func (s *store) PaymentInitiationsGet(ctx context.Context, piID models.PaymentInitiationID) (*models.PaymentInitiation, error) {
	var pi paymentInitiation
	err := s.db.NewSelect().
		Model(&pi).
		Where("id = ?", piID).
		Scan(ctx)
	if err != nil {
		return nil, e("failed to get payment initiation", err)
	}

	res := toPaymentInitiationModels(pi)
	return &res, nil
}

func (s *store) PaymentInitiationsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*paymentInitiation)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)
	return e("failed to delete payment initiations", err)
}

func (s *store) PaymentInitiationsDelete(ctx context.Context, piID models.PaymentInitiationID) error {
	_, err := s.db.NewDelete().
		Model((*paymentInitiation)(nil)).
		Where("id = ?", piID).
		Exec(ctx)
	return e("failed to delete payment initiation", err)
}

type PaymentInitiationQuery struct{}

type ListPaymentInitiationsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PaymentInitiationQuery]]

func NewListPaymentInitiationsQuery(opts bunpaginate.PaginatedQueryOptions[PaymentInitiationQuery]) ListPaymentInitiationsQuery {
	return ListPaymentInitiationsQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) paymentsInitiationQueryContext(qb query.Builder) (string, []any, error) {
	where, args, err := qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch {
		case key == "reference",
			key == "connector_id",
			key == "type",
			key == "asset",
			key == "source_account_id",
			key == "destination_account_id":
			if operator != "$match" {
				return "", nil, errors.Wrap(ErrValidation, "'type' column can only be used with $match")
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

func (s *store) PaymentInitiationsList(ctx context.Context, q ListPaymentInitiationsQuery) (*bunpaginate.Cursor[models.PaymentInitiation], error) {
	var (
		where string
		args  []any
		err   error
	)
	if q.Options.QueryBuilder != nil {
		where, args, err = s.paymentsInitiationQueryContext(q.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PaymentInitiationQuery], paymentInitiation](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PaymentInitiationQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			if where != "" {
				query = query.Where(where, args...)
			}

			// TODO(polo): sorter ?
			query = query.Order("created_at DESC")

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch accounts", err)
	}

	pis := make([]models.PaymentInitiation, 0, len(cursor.Data))
	for _, pi := range cursor.Data {
		pis = append(pis, toPaymentInitiationModels(pi))
	}

	return &bunpaginate.Cursor[models.PaymentInitiation]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     pis,
	}, nil
}

func (s *store) PaymentInitiationRelatedPaymentsUpsert(ctx context.Context, piID models.PaymentInitiationID, pID models.PaymentID, createdAt stdtime.Time) error {
	toInsert := paymentInitiationRelatedPayment{
		PaymentInitiationID: piID,
		PaymentID:           pID,
		CreatedAt:           time.New(createdAt),
	}

	_, err := s.db.NewInsert().
		Model(&toInsert).
		On("CONFLICT (payment_initiation_id, payment_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return e("failed to insert payment initiation related payments", err)
	}

	return nil
}

func (s *store) PaymentInitiationIDsListFromPaymentID(ctx context.Context, id models.PaymentID) ([]models.PaymentInitiationID, error) {
	var paymentInitiationRelatedPayments []paymentInitiationRelatedPayment
	err := s.db.NewSelect().
		Model(&paymentInitiationRelatedPayments).
		Column("payment_initiation_id").
		Where("payment_id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("failed to get payment initiation related payments", err)
	}

	ids := make([]models.PaymentInitiationID, 0, len(paymentInitiationRelatedPayments))
	for _, pi := range paymentInitiationRelatedPayments {
		ids = append(ids, pi.PaymentInitiationID)
	}

	return ids, nil
}

type PaymentInitiationRelatedPaymentsQuery struct{}

type ListPaymentInitiationRelatedPaymentsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PaymentInitiationRelatedPaymentsQuery]]

func NewListPaymentInitiationRelatedPaymentsQuery(opts bunpaginate.PaginatedQueryOptions[PaymentInitiationRelatedPaymentsQuery]) ListPaymentInitiationRelatedPaymentsQuery {
	return ListPaymentInitiationRelatedPaymentsQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) PaymentInitiationRelatedPaymentsList(ctx context.Context, piID models.PaymentInitiationID, q ListPaymentInitiationRelatedPaymentsQuery) (*bunpaginate.Cursor[models.Payment], error) {
	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PaymentInitiationRelatedPaymentsQuery], paymentInitiationRelatedPayment](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PaymentInitiationRelatedPaymentsQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			// TODO(polo): sorter ?
			query = query.Order("created_at DESC")
			query.Where("payment_initiation_id = ?", piID)

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch accounts", err)
	}

	pis := make([]models.Payment, 0, len(cursor.Data))
	for _, pi := range cursor.Data {
		p, err := s.PaymentsGet(ctx, pi.PaymentID)
		if err != nil {
			return nil, e("failed to get payment", err)
		}

		pis = append(pis, *p)
	}

	return &bunpaginate.Cursor[models.Payment]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     pis,
	}, nil
}

func (s *store) PaymentInitiationAdjustmentsUpsert(ctx context.Context, adj models.PaymentInitiationAdjustment) error {
	toInsert := fromPaymentInitiationAdjustmentModels(adj)

	_, err := s.db.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return e("failed to insert payment initiation adjustments", err)
	}

	return nil
}

func (s *store) PaymentInitiationAdjustmentsGet(ctx context.Context, id models.PaymentInitiationAdjustmentID) (*models.PaymentInitiationAdjustment, error) {
	var adj paymentInitiationAdjustment
	err := s.db.NewSelect().
		Model(&adj).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("failed to get payment initiation adjustment", err)
	}

	res := toPaymentInitiationAdjustmentModels(adj)
	return &res, nil
}

type PaymentInitiationAdjustmentsQuery struct{}

type ListPaymentInitiationAdjustmentsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PaymentInitiationAdjustmentsQuery]]

func NewListPaymentInitiationAdjustmentsQuery(opts bunpaginate.PaginatedQueryOptions[PaymentInitiationAdjustmentsQuery]) ListPaymentInitiationAdjustmentsQuery {
	return ListPaymentInitiationAdjustmentsQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) PaymentInitiationAdjustmentsList(ctx context.Context, piID models.PaymentInitiationID, q ListPaymentInitiationAdjustmentsQuery) (*bunpaginate.Cursor[models.PaymentInitiationAdjustment], error) {
	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PaymentInitiationAdjustmentsQuery], paymentInitiationAdjustment](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PaymentInitiationAdjustmentsQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			// TODO(polo): sorter ?
			query = query.Order("created_at DESC")
			query.Where("payment_initiation_id = ?", piID)

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch accounts", err)
	}

	pis := make([]models.PaymentInitiationAdjustment, 0, len(cursor.Data))
	for _, pi := range cursor.Data {
		pis = append(pis, toPaymentInitiationAdjustmentModels(pi))
	}

	return &bunpaginate.Cursor[models.PaymentInitiationAdjustment]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     pis,
	}, nil
}

func fromPaymentInitiationModels(from models.PaymentInitiation) paymentInitiation {
	return paymentInitiation{
		ID:                   from.ID,
		ConnectorID:          from.ConnectorID,
		Reference:            from.Reference,
		CreatedAt:            time.New(from.CreatedAt),
		ScheduledAt:          time.New(from.ScheduledAt),
		Description:          from.Description,
		Type:                 from.Type,
		Amount:               from.Amount,
		Asset:                from.Asset,
		DestinationAccountID: from.DestinationAccountID,
		SourceAccountID:      from.SourceAccountID,
		Metadata:             from.Metadata,
	}
}

func toPaymentInitiationModels(from paymentInitiation) models.PaymentInitiation {
	return models.PaymentInitiation{
		ID:                   from.ID,
		ConnectorID:          from.ConnectorID,
		Reference:            from.Reference,
		CreatedAt:            from.CreatedAt.Time,
		ScheduledAt:          from.ScheduledAt.Time,
		Description:          from.Description,
		Type:                 from.Type,
		SourceAccountID:      from.SourceAccountID,
		DestinationAccountID: from.DestinationAccountID,
		Amount:               from.Amount,
		Asset:                from.Asset,
		Metadata:             from.Metadata,
	}
}

func fromPaymentInitiationAdjustmentModels(from models.PaymentInitiationAdjustment) paymentInitiationAdjustment {
	return paymentInitiationAdjustment{
		ID:                  from.ID,
		PaymentInitiationID: from.PaymentInitiationID,
		CreatedAt:           time.New(from.CreatedAt),
		Status:              from.Status,
		Error: func() *string {
			if from.Error == nil {
				return nil
			}
			return pointer.For(from.Error.Error())
		}(),
		Metadata: from.Metadata,
	}
}

func toPaymentInitiationAdjustmentModels(from paymentInitiationAdjustment) models.PaymentInitiationAdjustment {
	return models.PaymentInitiationAdjustment{
		ID:                  from.ID,
		PaymentInitiationID: from.PaymentInitiationID,
		CreatedAt:           from.CreatedAt.Time,
		Status:              from.Status,
		Error: func() error {
			if from.Error == nil {
				return nil
			}

			return errors.New(*from.Error)
		}(),
		Metadata: from.Metadata,
	}
}
