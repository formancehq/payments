package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgxlisten"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

type ConnectorChanges string

const (
	ConnectorChangesInsert ConnectorChanges = "insert_"
	ConnectorChangesUpdate ConnectorChanges = "update_"
	ConnectorChangesDelete ConnectorChanges = "delete_"
)

type HandlerConnectorsChanges map[ConnectorChanges]func(context.Context, models.ConnectorID) error

type connector struct {
	bun.BaseModel `bun:"table:connectors"`

	// Mandatory fields
	ID                   models.ConnectorID `bun:"id,pk,type:character varying,notnull"`
	Reference            uuid.UUID          `bun:"reference,type:uuid,notnull"`
	Name                 string             `bun:"name,type:text,notnull"`
	CreatedAt            time.Time          `bun:"created_at,type:timestamp without time zone,notnull"`
	Provider             string             `bun:"provider,type:text,notnull"`
	ScheduledForDeletion bool               `bun:"scheduled_for_deletion,type:boolean,notnull"`

	// EncryptedConfig is a PGP-encrypted JSON string.
	EncryptedConfig string `bun:"config,type:bytea,notnull"`

	// Config is a decrypted config. It is not stored in the database.
	DecryptedConfig json.RawMessage `bun:"decrypted_config,scanonly"`
}

func (s *store) ListenConnectorsChanges(ctx context.Context, handlers HandlerConnectorsChanges) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("cannot get connection: %w", err)
	}

	s.rwMutex.Lock()
	s.conns = append(s.conns, conn)
	s.rwMutex.Unlock()

	if err := conn.Raw(func(driverConn any) error {
		listener := pgxlisten.Listener{
			Connect: func(ctx context.Context) (*pgx.Conn, error) {
				return pgx.Connect(ctx, driverConn.(*stdlib.Conn).Conn().Config().ConnString())
			},
		}
		listener.Handle("connectors", pgxlisten.HandlerFunc(func(ctx context.Context, notification *pgconn.Notification, conn *pgx.Conn) error {
			for prefix, handler := range handlers {
				if strings.HasPrefix(notification.Payload, string(prefix)) {
					return handler(
						ctx,
						models.MustConnectorIDFromString(strings.TrimPrefix(notification.Payload, string(prefix))),
					)
				}
			}
			return nil
		}))
		go func() {
			s.logger.Info("listening for connectors changes")
			if err := listener.Listen(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}

				s.logger.Errorf("failed to listen for connectors changes: %w", err)
			}
		}()
		return nil
	}); err != nil {
		return fmt.Errorf("cannot get driver connection: %w", err)
	}
	return nil
}

func (s *store) ConnectorsInstall(ctx context.Context, c models.Connector, oldConnectorID *models.ConnectorID) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	toInsert := connector{
		ID:                   c.ID,
		Reference:            c.ID.Reference,
		Name:                 c.Name,
		CreatedAt:            time.New(c.CreatedAt),
		Provider:             c.Provider,
		ScheduledForDeletion: false,
	}

	_, err = tx.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return e("failed to insert connector", err)
	}

	_, err = tx.NewUpdate().
		Model((*connector)(nil)).
		Set("config = pgp_sym_encrypt(?::TEXT, ?, ?)", c.Config, s.configEncryptionKey, encryptionOptions).
		Where("id = ?", toInsert.ID).
		Exec(ctx)
	if err != nil {
		return e("failed to encrypt config", err)
	}

	// Create outbox event for connector reset if oldConnectorID is provided
	if oldConnectorID != nil {
		now := toInsert.CreatedAt.UTC()
		payload := map[string]interface{}{
			"createdAt":   now.Time,
			"connectorID": oldConnectorID.String(),
		}

		var payloadBytes []byte
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return e("failed to marshal connector reset event payload", err)
		}

		idempotencyKey := fmt.Sprintf("%s-%s", oldConnectorID.String(), now.Time.Format(time.RFC3339Nano))
		outboxEvent := models.OutboxEvent{
			EventType:      models.OUTBOX_EVENT_CONNECTOR_RESET,
			EntityID:       oldConnectorID.String(),
			Payload:        payloadBytes,
			CreatedAt:      now.Time,
			Status:         models.OUTBOX_STATUS_PENDING,
			ConnectorID:    &toInsert.ID,
			IdempotencyKey: idempotencyKey,
		}

		if err = s.OutboxEventsInsert(ctx, tx, []models.OutboxEvent{outboxEvent}); err != nil {
			return err
		}
	}

	return e("failed to commit transaction", tx.Commit())
}

func (s *store) ConnectorsConfigUpdate(ctx context.Context, c models.Connector) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	_, err = s.ConnectorsGet(ctx, c.ID)
	if err != nil {
		return e("connector not found", err)
	}

	_, err = tx.NewUpdate().
		Model((*connector)(nil)).
		Set("name = ?", c.Name).
		Set("config = pgp_sym_encrypt(?::TEXT, ?, ?)", c.Config, s.configEncryptionKey, encryptionOptions).
		Where("id = ?", c.ID).
		Exec(ctx)
	if err != nil {
		return e("failed to encrypt config", err)
	}

	return e("failed to commit transaction", tx.Commit())
}

func (s *store) ConnectorsScheduleForDeletion(ctx context.Context, id models.ConnectorID) error {
	_, err := s.db.NewUpdate().
		Model((*connector)(nil)).
		Set("scheduled_for_deletion = ?", true).
		Where("id = ?", id).
		Exec(ctx)
	return e("failed to schedule connector for deletion", err)
}

func (s *store) ConnectorsUninstall(ctx context.Context, id models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*connector)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	return e("failed to delete connector", err)
}

func (s *store) ConnectorsGet(ctx context.Context, id models.ConnectorID) (*models.Connector, error) {
	var connector connector

	err := s.db.NewSelect().
		Model(&connector).
		ColumnExpr("*, pgp_sym_decrypt(config, ?, ?) AS decrypted_config", s.configEncryptionKey, encryptionOptions).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("failed to fetch connector", err)
	}

	return &models.Connector{
		ConnectorBase: models.ConnectorBase{
			ID:        connector.ID,
			Name:      connector.Name,
			CreatedAt: connector.CreatedAt.Time,
			Provider:  connector.Provider,
		},
		Config:               connector.DecryptedConfig,
		ScheduledForDeletion: connector.ScheduledForDeletion,
	}, nil
}

type ConnectorQuery struct{}

type ListConnectorsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[ConnectorQuery]]

func NewListConnectorsQuery(opts bunpaginate.PaginatedQueryOptions[ConnectorQuery]) ListConnectorsQuery {
	return ListConnectorsQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) connectorsQueryContext(qb query.Builder) (string, []any, error) {
	return qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch key {
		case "provider":
			v, ok := value.(string)
			if !ok {
				return "", nil, fmt.Errorf("expected string type for provider, got %T: %w", value, ErrValidation)
			}
			return fmt.Sprintf("%s %s ?", key, query.DefaultComparisonOperatorsMapping[operator]), []any{strings.ToLower(models.ToV3Provider(v))}, nil
		case "name", "id":
			return fmt.Sprintf("%s %s ?", key, query.DefaultComparisonOperatorsMapping[operator]), []any{value}, nil
		default:
			return "", nil, fmt.Errorf("unknown key '%s' when building query: %w", key, ErrValidation)
		}
	}))
}

func (s *store) ConnectorsList(ctx context.Context, q ListConnectorsQuery) (*bunpaginate.Cursor[models.Connector], error) {
	var (
		where string
		args  []any
		err   error
	)
	if q.Options.QueryBuilder != nil {
		where, args, err = s.connectorsQueryContext(q.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[ConnectorQuery], connector](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[ConnectorQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			if where != "" {
				query = query.Where(where, args...)
			}

			query = query.ColumnExpr("*, pgp_sym_decrypt(config, ?, ?) AS decrypted_config", s.configEncryptionKey, encryptionOptions)

			// TODO(polo): sorter ?
			query = query.Order("created_at DESC", "sort_id DESC")

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch connectors", err)
	}

	connectors := make([]models.Connector, 0, len(cursor.Data))
	for _, c := range cursor.Data {
		connectors = append(connectors, toConnectorModels(c))
	}

	return &bunpaginate.Cursor[models.Connector]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     connectors,
	}, nil
}

func toConnectorModels(from connector) models.Connector {
	return models.Connector{
		ConnectorBase:        toConnectorBaseModels(from),
		Config:               from.DecryptedConfig,
		ScheduledForDeletion: from.ScheduledForDeletion,
	}
}

func toConnectorBaseModels(from connector) models.ConnectorBase {
	return models.ConnectorBase{
		ID:        from.ID,
		Name:      from.Name,
		CreatedAt: from.CreatedAt.Time,
		Provider:  from.Provider,
	}
}
