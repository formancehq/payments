package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	internalTime "github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
)

type conversion struct {
	bun.BaseModel `bun:"table:conversions"`

	// Mandatory fields
	ID           models.ConversionID     `bun:"id,pk,type:character varying,notnull"`
	ConnectorID  models.ConnectorID      `bun:"connector_id,type:character varying,notnull"`
	Reference    string                  `bun:"reference,type:text,notnull"`
	CreatedAt    internalTime.Time       `bun:"created_at,type:timestamp without time zone,notnull"`
	UpdatedAt    internalTime.Time       `bun:"updated_at,type:timestamp without time zone,notnull"`
	SourceAsset  string                  `bun:"source_asset,type:text,notnull"`
	DestinationAsset  string                  `bun:"destination_asset,type:text,notnull"`
	SourceAmount *big.Int                `bun:"source_amount,type:numeric,notnull"`
	Status       models.ConversionStatus `bun:"status,type:text,notnull"`

	// Optional fields
	DestinationAmount    *big.Int `bun:"destination_amount,type:numeric,nullzero"`
	Fee                  *big.Int `bun:"fee,type:numeric,nullzero"`
	FeeAsset             *string  `bun:"fee_asset,type:text,nullzero"`
	SourceAccountID      *models.AccountID `bun:"source_account_id,type:character varying,nullzero"`
	DestinationAccountID *models.AccountID `bun:"destination_account_id,type:character varying,nullzero"`

	// Optional fields with default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`

	// Raw PSP response
	Raw json.RawMessage `bun:"raw,type:json,notnull"`
}

func (s *store) ConversionsUpsert(ctx context.Context, conversions []models.Conversion) error {
	conversionsToInsert := make([]conversion, 0, len(conversions))

	for _, c := range conversions {
		conversionsToInsert = append(conversionsToInsert, fromConversionModels(c))
	}

	if len(conversionsToInsert) > 0 {
		_, err := s.db.NewInsert().
			Model(&conversionsToInsert).
			On("CONFLICT (id) DO UPDATE").
			Set("updated_at = EXCLUDED.updated_at").
			Set("status = EXCLUDED.status").
			Set("destination_amount = EXCLUDED.destination_amount").
			Set("fee = EXCLUDED.fee").
			Set("fee_asset = EXCLUDED.fee_asset").
			Set("source_account_id = EXCLUDED.source_account_id").
			Set("destination_account_id = EXCLUDED.destination_account_id").
			Set("metadata = conversion.metadata || EXCLUDED.metadata").
			Exec(ctx)
		if err != nil {
			return e("failed to insert conversions", err)
		}
	}

	return nil
}

func (s *store) ConversionsGet(ctx context.Context, id models.ConversionID) (*models.Conversion, error) {
	var c conversion
	err := s.db.NewSelect().
		Model(&c).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("failed to get conversion", err)
	}

	res := toConversionModels(c)
	return &res, nil
}

func (s *store) ConversionsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*conversion)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)

	return e("failed to delete conversions", err)
}

type ConversionQuery struct{}

type ListConversionsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[ConversionQuery]]

func NewListConversionsQuery(opts bunpaginate.PaginatedQueryOptions[ConversionQuery]) ListConversionsQuery {
	return ListConversionsQuery{
		PageSize: opts.PageSize,
		Order:    bunpaginate.OrderAsc,
		Options:  opts,
	}
}

func (s *store) conversionsQueryContext(qb query.Builder) (string, []any, error) {
	where, args, err := qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch {
		case key == "reference",
			key == "id",
			key == "connector_id",
			key == "source_asset",
			key == "destination_asset",
			key == "status",
			key == "source_account_id",
			key == "destination_account_id":
			if operator != "$match" {
				return "", nil, e(fmt.Sprintf("'%s' column can only be used with $match", key), ErrValidation)
			}
			return fmt.Sprintf("conversion.%s = ?", key), []any{value}, nil

		case key == "source_amount",
			key == "destination_amount":
			return fmt.Sprintf("conversion.%s %s ?", key, query.DefaultComparisonOperatorsMapping[operator]), []any{value}, nil
		case metadataRegex.Match([]byte(key)):
			if operator != "$match" {
				return "", nil, e("'metadata' column can only be used with $match", ErrValidation)
			}
			match := metadataRegex.FindAllStringSubmatch(key, 3)

			return "conversion.metadata @> ?", []any{map[string]any{
				match[0][1]: value,
			}}, nil
		default:
			return "", nil, fmt.Errorf("unknown key '%s' when building query: %w", key, ErrValidation)
		}
	}))

	return where, args, err
}

func (s *store) ConversionsList(ctx context.Context, q ListConversionsQuery) (*bunpaginate.Cursor[models.Conversion], error) {
	var (
		where string
		args  []any
		err   error
	)
	if q.Options.QueryBuilder != nil {
		where, args, err = s.conversionsQueryContext(q.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[ConversionQuery], conversion](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[ConversionQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			if where != "" {
				query = query.Where(where, args...)
			}

			query = query.Order("created_at DESC", "sort_id DESC")

			return query
		},
	)
	if err != nil {
		return nil, err
	}

	conversions := make([]models.Conversion, 0, len(cursor.Data))
	for _, c := range cursor.Data {
		conversions = append(conversions, toConversionModels(c))
	}

	return &bunpaginate.Cursor[models.Conversion]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     conversions,
	}, nil
}

func fromConversionModels(from models.Conversion) conversion {
	return conversion{
		ID:                   from.ID,
		ConnectorID:          from.ConnectorID,
		Reference:            from.Reference,
		CreatedAt:            internalTime.New(from.CreatedAt),
		UpdatedAt:            internalTime.New(from.UpdatedAt),
		SourceAsset:          from.SourceAsset,
		DestinationAsset:     from.DestinationAsset,
		SourceAmount:         from.SourceAmount,
		DestinationAmount:    from.DestinationAmount,
		Fee:                  from.Fee,
		FeeAsset:             from.FeeAsset,
		Status:               from.Status,
		SourceAccountID:      from.SourceAccountID,
		DestinationAccountID: from.DestinationAccountID,
		Metadata:             from.Metadata,
		Raw:                  from.Raw,
	}
}

func toConversionModels(from conversion) models.Conversion {
	return models.Conversion{
		ID:                   from.ID,
		ConnectorID:          from.ConnectorID,
		Reference:            from.Reference,
		CreatedAt:            from.CreatedAt.Time,
		UpdatedAt:            from.UpdatedAt.Time,
		SourceAsset:          from.SourceAsset,
		DestinationAsset:     from.DestinationAsset,
		SourceAmount:         from.SourceAmount,
		DestinationAmount:    from.DestinationAmount,
		Fee:                  from.Fee,
		FeeAsset:             from.FeeAsset,
		Status:               from.Status,
		SourceAccountID:      from.SourceAccountID,
		DestinationAccountID: from.DestinationAccountID,
		Metadata:             from.Metadata,
		Raw:                  from.Raw,
	}
}
