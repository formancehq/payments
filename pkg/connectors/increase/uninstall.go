package increase

import (
	"context"
	"strings"

	"github.com/formancehq/payments/pkg/connectors/increase/client"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) uninstall(ctx context.Context, req connector.UninstallRequest) (connector.UninstallResponse, error) {
	webhooks, err := p.client.ListEventSubscriptions(ctx)
	if err != nil {
		return connector.UninstallResponse{}, err
	}

	for _, webhook := range webhooks {
		if !strings.Contains(webhook.URL, req.ConnectorID) {
			continue
		}

		es := &client.UpdateEventSubscriptionRequest{
			Status: eventSubscriptionStatusDeleted,
		}

		if _, err := p.client.UpdateEventSubscription(ctx, es, webhook.ID); err != nil {
			return connector.UninstallResponse{}, err
		}
	}

	return connector.UninstallResponse{}, nil
}
