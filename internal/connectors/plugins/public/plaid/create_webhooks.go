package plaid

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

type supportedWebhook struct {
	urlPath string
	fn      func(context.Context, models.TranslateWebhookRequest) ([]models.WebhookResponse, error)
}

func (p *Plugin) initWebhookConfig() {
	p.supportedWebhooks = map[string]supportedWebhook{
		"all": supportedWebhook{
			urlPath: "/all",
			fn:      p.handleAllWebhook,
		},
	}
}

func (p *Plugin) createWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	configs := make([]models.PSPWebhookConfig, 0, len(p.supportedWebhooks))
	for name, w := range p.supportedWebhooks {
		configs = append(configs, models.PSPWebhookConfig{
			Name:    name,
			URLPath: w.urlPath,
		})
	}

	return models.CreateWebhooksResponse{
		Configs: configs,
	}, nil
}
