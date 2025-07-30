package storage

import (
	"context"
	"github.com/formancehq/go-libs/v3/platform/postgres"
	"github.com/pkg/errors"

	"github.com/formancehq/payments/internal/models"
	"github.com/uptrace/bun"
)

type webhookConfig struct {
	bun.BaseModel `bun:"table:webhooks_configs"`

	// Mandatory fields
	Name        string             `bun:"name,pk,type:text,notnull"`
	ConnectorID models.ConnectorID `bun:"connector_id,pk,type:character varying,notnull"`
	URLPath     string             `bun:"url_path,type:text,notnull"`

	// Optional fields with default
	// c.f. https://bun.uptrace.dev/guide/models.html#default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

func (s *store) WebhooksConfigsUpsert(ctx context.Context, webhooksConfigs []models.WebhookConfig) error {
	toInsert := fromWebhooksConfigsModels(webhooksConfigs)

	_, err := s.db.NewInsert().
		Model(&toInsert).
		On("CONFLICT (name, connector_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return errors.Wrap(postgres.ResolveError(err), "upsert webhook config")
	}

	return nil
}

func (s *store) WebhooksConfigsGet(ctx context.Context, name string, connectorID models.ConnectorID) (*models.WebhookConfig, error) {
	var webhookConfig webhookConfig
	err := s.db.NewSelect().
		Model(&webhookConfig).
		Where("name = ? AND connector_id = ?", name, connectorID).
		Scan(ctx)
	if err != nil {
		return nil, errors.Wrap(postgres.ResolveError(err), "get webhook config")
	}

	return toWebhookConfigModel(webhookConfig), nil
}

func (s *store) WebhooksConfigsGetFromConnectorID(ctx context.Context, connectorID models.ConnectorID) ([]models.WebhookConfig, error) {
	var webhookConfigs []webhookConfig
	err := s.db.NewSelect().
		Model(&webhookConfigs).
		Where("connector_id = ?", connectorID).
		Scan(ctx)
	if err != nil {
		return nil, errors.Wrap(postgres.ResolveError(err), "get webhook configs")
	}

	res := make([]models.WebhookConfig, 0, len(webhookConfigs))
	for _, webhookConfig := range webhookConfigs {
		res = append(res, *toWebhookConfigModel(webhookConfig))
	}

	return res, nil
}

func (s *store) WebhooksConfigsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*webhookConfig)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)
	if err != nil {
		return errors.Wrap(postgres.ResolveError(err), "delete webhook config")
	}

	return nil
}

func fromWebhookConfigModels(from models.WebhookConfig) webhookConfig {
	return webhookConfig{
		Name:        from.Name,
		ConnectorID: from.ConnectorID,
		URLPath:     from.URLPath,
		Metadata:    from.Metadata,
	}
}

func fromWebhooksConfigsModels(from []models.WebhookConfig) []webhookConfig {
	to := make([]webhookConfig, 0, len(from))
	for _, webhookConfig := range from {
		to = append(to, fromWebhookConfigModels(webhookConfig))
	}

	return to
}

func toWebhookConfigModel(from webhookConfig) *models.WebhookConfig {
	return &models.WebhookConfig{
		Name:        from.Name,
		ConnectorID: from.ConnectorID,
		URLPath:     from.URLPath,
		Metadata:    from.Metadata,
	}
}
