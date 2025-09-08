package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type account struct {
	bun.BaseModel `bun:"table:accounts"`

	// Mandatory fields
	ID          models.AccountID   `bun:"id,pk,type:character varying,notnull"`
	ConnectorID models.ConnectorID `bun:"connector_id,type:character varying,notnull"`
	CreatedAt   time.Time          `bun:"created_at,type:timestamp without time zone,notnull"`
	Reference   string             `bun:"reference,type:text,notnull"`
	Type        string             `bun:"type,type:text,notnull"`
	Raw         json.RawMessage    `bun:"raw,type:json,notnull"`

	// Optional fields
	// c.f.: https://bun.uptrace.dev/guide/models.html#nulls
	DefaultAsset            *string    `bun:"default_asset,type:text,nullzero"`
	Name                    *string    `bun:"name,type:text,nullzero"`
	PsuID                   *uuid.UUID `bun:"psu_id,type:uuid,nullzero"`
	OpenBankingConnectionID *string    `bun:"open_banking_connection_id,type:character varying,nullzero"`

	// Optional fields with default
	// c.f. https://bun.uptrace.dev/guide/models.html#default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

func (s *store) AccountsUpsert(ctx context.Context, accounts []models.Account) error {
	if len(accounts) == 0 {
		return nil
	}

	toInsert := make([]account, 0, len(accounts))
	for _, a := range accounts {
		toInsert = append(toInsert, fromAccountModels(a))
	}

	_, err := s.db.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)

	return e("failed to insert accounts", err)
}

func (s *store) AccountsGet(ctx context.Context, id models.AccountID) (*models.Account, error) {
	var account account

	err := s.db.NewSelect().
		Model(&account).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("failed to get account", err)
	}

	res := toAccountModels(account)
	return &res, nil
}

func (s *store) AccountsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*account)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)

	return e("failed to delete account", err)
}

func (s *store) AccountsDeleteFromPSUID(ctx context.Context, psuID uuid.UUID) error {
	_, err := s.db.NewDelete().
		Model((*account)(nil)).
		Where("psu_id = ?", psuID).
		Exec(ctx)

	return e("failed to delete account", err)
}

func (s *store) AccountsDeleteFromConnectorIDAndPSUID(ctx context.Context, connectorID models.ConnectorID, psuID uuid.UUID) error {
	_, err := s.db.NewDelete().
		Model((*account)(nil)).
		Where("connector_id = ?", connectorID).
		Where("psu_id = ?", psuID).
		Exec(ctx)

	return e("failed to delete account", err)
}

func (s *store) AccountsDeleteFromOpenBankingConnectionID(ctx context.Context, psuID uuid.UUID, openBankingConnectionID string) error {
	_, err := s.db.NewDelete().
		Model((*account)(nil)).
		Where("psu_id = ?", psuID).
		Where("open_banking_connection_id = ?", openBankingConnectionID).
		Exec(ctx)

	return e("failed to delete account", err)
}

// TODO(polo): add tests
func (s *store) AccountsDelete(ctx context.Context, id models.AccountID) error {
	_, err := s.db.NewDelete().
		Model((*account)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return e("failed to delete account", err)
}

type AccountQuery struct{}

type ListAccountsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[AccountQuery]]

func NewListAccountsQuery(opts bunpaginate.PaginatedQueryOptions[AccountQuery]) ListAccountsQuery {
	return ListAccountsQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) accountsQueryContext(qb query.Builder) (string, []any, error) {
	return qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch {
		case key == "id",
			key == "reference",
			key == "connector_id",
			key == "type",
			key == "default_asset",
			key == "name",
			key == "psu_id",
			key == "open_banking_connection_id":
			return fmt.Sprintf("%s %s ?", key, query.DefaultComparisonOperatorsMapping[operator]), []any{value}, nil
		case metadataRegex.Match([]byte(key)):
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
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
}

func (s *store) AccountsList(ctx context.Context, q ListAccountsQuery) (*bunpaginate.Cursor[models.Account], error) {
	var (
		where string
		args  []any
		err   error
	)
	if q.Options.QueryBuilder != nil {
		where, args, err = s.accountsQueryContext(q.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[AccountQuery], account](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[AccountQuery]])(&q),
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
		return nil, e("failed to fetch accounts", err)
	}

	accounts := make([]models.Account, 0, len(cursor.Data))
	for _, a := range cursor.Data {
		accounts = append(accounts, toAccountModels(a))
	}

	return &bunpaginate.Cursor[models.Account]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     accounts,
	}, nil
}

func fromAccountModels(from models.Account) account {
	return account{
		ID:                      from.ID,
		ConnectorID:             from.ConnectorID,
		CreatedAt:               time.New(from.CreatedAt),
		Reference:               from.Reference,
		Type:                    string(from.Type),
		DefaultAsset:            from.DefaultAsset,
		Name:                    from.Name,
		PsuID:                   from.PsuID,
		OpenBankingConnectionID: from.OpenBankingConnectionID,
		Metadata:                from.Metadata,
		Raw:                     from.Raw,
	}
}

func toAccountModels(from account) models.Account {
	return models.Account{
		ID:                      from.ID,
		ConnectorID:             from.ConnectorID,
		Reference:               from.Reference,
		CreatedAt:               from.CreatedAt.Time,
		Type:                    models.AccountType(from.Type),
		Name:                    from.Name,
		DefaultAsset:            from.DefaultAsset,
		PsuID:                   from.PsuID,
		OpenBankingConnectionID: from.OpenBankingConnectionID,
		Metadata:                from.Metadata,
		Raw:                     from.Raw,
	}
}
