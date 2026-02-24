package column

import (
	"context"
	"strings"

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
		if _, err := p.client.DeleteEventSubscription(ctx, webhook.ID); err != nil {
			return connector.UninstallResponse{}, err
		}
	}
	return connector.UninstallResponse{}, nil
}
