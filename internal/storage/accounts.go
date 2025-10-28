package storage

import (
	"context"
	"database/sql"
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

	Connector *connector `bun:"rel:belongs-to,join:connector_id=id,alt:connector_"`
}

func (s *store) AccountsUpsert(ctx context.Context, accounts []models.Account) error {
	if len(accounts) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	toInsert := make([]account, 0, len(accounts))
	for _, a := range accounts {
		acc := fromAccountModels(a)
		toInsert = append(toInsert, acc)
	}

	// Insert accounts with ON CONFLICT DO NOTHING and capture inserted rows
	var insertedAccounts []account
	err = tx.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id) DO NOTHING").
		Returning("*").
		Scan(ctx, &insertedAccounts)
	if err != nil {
		return e("failed to insert accounts", err)
	}

	// Create a map of inserted account IDs for quick lookup
	insertedAccountIDs := make(map[string]bool)
	for _, insertedAccount := range insertedAccounts {
		insertedAccountIDs[insertedAccount.ID.String()] = true
	}

	// Create outbox events only for newly inserted accounts
	outboxEvents := make([]models.OutboxEvent, 0, len(insertedAccounts))
	for _, account := range accounts {
		// Skip accounts that already existed (not in the inserted set)
		if !insertedAccountIDs[account.ID.String()] {
			continue
		}
		// Create the event payload
		payload := map[string]interface{}{
			"id":          account.ID.String(),
			"connectorID": account.ConnectorID.String(),
			"provider":    models.ToV3Provider(account.ConnectorID.Provider),
			"createdAt":   account.CreatedAt,
			"reference":   account.Reference,
			"type":        string(account.Type),
			"metadata":    account.Metadata,
			"rawData":     account.Raw,
		}

		if account.DefaultAsset != nil {
			payload["defaultAsset"] = *account.DefaultAsset
		}

		if account.Name != nil {
			payload["name"] = *account.Name
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal account event payload: %w", err)
		}

		cid := account.ConnectorID
		outboxEvent := models.OutboxEvent{
			EventType:      "account.saved",
			EntityID:       account.ID.String(),
			Payload:        payloadBytes,
			CreatedAt:      time.Now().UTC().Time,
			Status:         models.OUTBOX_STATUS_PENDING,
			ConnectorID:    &cid,
			IdempotencyKey: account.IdempotencyKey(),
		}

		outboxEvents = append(outboxEvents, outboxEvent)
	}

	// Insert outbox events in the same transaction
	if len(outboxEvents) > 0 {
		if err := s.OutboxEventsInsert(ctx, tx, outboxEvents); err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return e("failed to commit transaction", err)
	}
	return nil
}

func (s *store) AccountsGet(ctx context.Context, id models.AccountID) (*models.Account, error) {
	var account account

	err := s.db.NewSelect().
		Model(&account).
		Where("account.id = ?", id).
		Relation("Connector").
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

func (s *store) AccountsDeleteFromOpenBankingConnectionID(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, openBankingConnectionID string) error {
	_, err := s.db.NewDelete().
		Model((*account)(nil)).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
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
			return fmt.Sprintf("account.%s %s ?", key, query.DefaultComparisonOperatorsMapping[operator]), []any{value}, nil
		case metadataRegex.Match([]byte(key)):
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}
			match := metadataRegex.FindAllStringSubmatch(key, 3)

			key := "account.metadata"
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
			query = query.Relation("Connector")

			// TODO(polo): sorter ?
			query = query.Order("account.created_at DESC", "account.sort_id DESC")

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
	acc := models.Account{
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

	if from.Connector != nil {
		c := toConnectorBaseModels(*from.Connector)
		acc.Connector = &c
	}
	return acc
}
