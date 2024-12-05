package migrations

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
)

const (
	encryptionOptions = "compress-algo=1, cipher-algo=aes256"
)

type v2Connector struct {
	bun.BaseModel `bun:"connectors.connector"`
	ID            models.ConnectorID `bun:"id,pk,nullzero"`
	Name          string             `bun:"name"`
	CreatedAt     time.Time          `bun:"created_at,nullzero"`
	Provider      string
	// EncryptedConfig is a PGP-encrypted JSON string.
	EncryptedConfig string `bun:"config"`
	// Config is a decrypted config. It is not stored in the database.
	Config json.RawMessage `bun:"decrypted_config,scanonly"`
}

type v3Connector struct {
	bun.BaseModel        `bun:"table:connectors"`
	ID                   models.ConnectorID `bun:"id,pk,type:character varying,notnull"`
	Name                 string             `bun:"name,type:text,notnull"`
	CreatedAt            time.Time          `bun:"created_at,type:timestamp without time zone,notnull"`
	Provider             string             `bun:"provider,type:text,notnull"`
	ScheduledForDeletion bool               `bun:"scheduled_for_deletion,type:boolean,notnull"`
	EncryptedConfig      string             `bun:"config,type:bytea,notnull"`
	DecryptedConfig      json.RawMessage    `bun:"decrypted_config,scanonly"`
}

type ConnectorQuery struct{}

func MigrateConnectorsFromV2(ctx context.Context, db bun.IDB, encryptionKey string) error {
	exist, err := isTableExisting(ctx, db, "connectors", "connector")
	if err != nil {
		return err
	}

	if !exist {
		// Nothing to migrate
		return nil
	}

	_, err = db.ExecContext(ctx, `
		ALTER TABLE connectors.connector ADD COLUMN IF NOT EXISTS sort_id bigserial;
	`)
	if err != nil {
		return err
	}

	q := bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[ConnectorQuery]]{
		Order:    bunpaginate.OrderAsc,
		PageSize: 100,
		Options: bunpaginate.PaginatedQueryOptions[ConnectorQuery]{
			PageSize: 100,
			Options:  ConnectorQuery{},
		},
	}
	for {
		cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[ConnectorQuery], v2Connector](
			ctx,
			db,
			(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[ConnectorQuery]])(&q),
			func(query *bun.SelectQuery) *bun.SelectQuery {
				return query.
					ColumnExpr("*, pgp_sym_decrypt(config, ?, ?) AS decrypted_config", encryptionKey, encryptionOptions).
					Order("created_at ASC", "sort_id ASC")
			},
		)
		if err != nil {
			return err
		}

		for _, connector := range cursor.Data {
			v3 := v3Connector{
				ID:                   connector.ID,
				Name:                 connector.Name,
				CreatedAt:            connector.CreatedAt,
				Provider:             connector.Provider,
				ScheduledForDeletion: false,
			}

			shouldInsert, v3Config, err := transformV2ConfigToV3Config(connector.Provider, connector.Config)
			if err != nil {
				return err
			}

			if !shouldInsert {
				continue
			}

			_, err = db.NewInsert().
				Model(&v3).
				On("conflict (id) do nothing").
				Exec(ctx)
			if err != nil {
				return err
			}

			_, err = db.NewUpdate().
				Model((*v3Connector)(nil)).
				Set("config = pgp_sym_encrypt(?::TEXT, ?, ?)", v3Config, encryptionKey, encryptionOptions).
				Where("id = ?", v3.ID).
				Exec(ctx)
			if err != nil {
				return err
			}
		}

		if !cursor.HasMore {
			break
		}

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		if err != nil {
			return err
		}
	}

	return nil
}
