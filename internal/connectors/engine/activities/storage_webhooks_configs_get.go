package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageWebhooksConfigsGet(ctx context.Context, connectorID models.ConnectorID) ([]models.WebhookConfig, error) {
	configs, err := a.storage.WebhooksConfigsGetFromConnectorID(ctx, connectorID)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return configs, nil
}

var StorageWebhooksConfigsGetActivity = Activities{}.StorageWebhooksConfigsGet

func StorageWebhooksConfigsGet(ctx workflow.Context, connectorID models.ConnectorID) ([]models.WebhookConfig, error) {
	var res []models.WebhookConfig
	err := executeActivity(ctx, StorageWebhooksConfigsGetActivity, &res, connectorID)
	return res, err
}
