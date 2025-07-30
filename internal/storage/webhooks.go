package storage

import (
	"context"
	"github.com/formancehq/go-libs/v3/platform/postgres"
	"github.com/pkg/errors"

	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
)

type webhook struct {
	bun.BaseModel `bun:"table:webhooks"`

	// Mandatory fields
	ID          string             `bun:"id,pk,type:uuid,notnull"`
	ConnectorID models.ConnectorID `bun:"connector_id,type:character varying,notnull"`

	// Optional fields
	IdempotencyKey *string             `bun:"idempotency_key,type:text"`
	Headers        map[string][]string `bun:"headers,type:json"`
	QueryValues    map[string][]string `bun:"query_values,type:json"`
	Body           []byte              `bun:"body,type:bytea,nullzero"`
}

func (s *store) WebhooksInsert(ctx context.Context, webhook models.Webhook) error {
	toInsert := fromWebhookModels(webhook)

	_, err := s.db.NewInsert().
		Model(&toInsert).
		On("CONFLICT (id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return errors.Wrap(postgres.ResolveError(err), "insert webhook")
	}

	return nil
}

func (s *store) WebhooksGet(ctx context.Context, id string) (models.Webhook, error) {
	var w webhook
	err := s.db.NewSelect().
		Model(&w).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return models.Webhook{}, errors.Wrap(postgres.ResolveError(err), "get webhook")
	}

	return toWebhookModels(w), nil
}

func (s *store) WebhooksDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*webhook)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)
	if err != nil {
		return errors.Wrap(postgres.ResolveError(err), "delete webhook")
	}

	return nil
}

func fromWebhookModels(from models.Webhook) webhook {
	return webhook{
		ID:             from.ID,
		ConnectorID:    from.ConnectorID,
		IdempotencyKey: from.IdempotencyKey,
		Headers:        from.Headers,
		QueryValues:    from.QueryValues,
		Body:           from.Body,
	}
}

func toWebhookModels(from webhook) models.Webhook {
	return models.Webhook{
		ID:             from.ID,
		ConnectorID:    from.ConnectorID,
		IdempotencyKey: from.IdempotencyKey,
		Headers:        from.Headers,
		QueryValues:    from.QueryValues,
		Body:           from.Body,
	}
}
