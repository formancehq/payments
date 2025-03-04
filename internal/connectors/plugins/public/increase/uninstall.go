package increase

import (
	"context"
	"strings"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	webhooks, err := p.client.ListEventSubscriptions(ctx)
	if err != nil {
		return models.UninstallResponse{}, err
	}

	for _, webhook := range webhooks {
		if !strings.Contains(webhook.URL, req.ConnectorID) {
			continue
		}

		es := &client.UpdateEventSubscriptionRequest{
			Status: eventSubscriptionStatusDeleted,
		}

		if _, err := p.client.UpdateEventSubscription(ctx, es, webhook.ID); err != nil {
			return models.UninstallResponse{}, err
		}
	}

	return models.UninstallResponse{}, nil
}
