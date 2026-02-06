package adyen

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connectors/adyen/client"
	"github.com/formancehq/payments/pkg/registry"
)

const ProviderName = "adyen"

func init() {
	registry.RegisterPlugin(ProviderName, connector.PluginTypePSP, func(_ connector.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (connector.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
	connector.Plugin

	name string

	logger            logging.Logger
	client            client.Client
	config            Config
	supportedWebhooks map[string]supportedWebhook
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := client.New(
		ProviderName,
		config.APIKey,
		config.WebhookUsername,
		config.WebhookPassword,
		config.CompanyID,
		config.LiveEndpointPrefix,
	)

	p := &Plugin{
		Plugin: connector.NewBasePlugin(),

		name:   name,
		logger: logger,
		client: client,
		config: config,
	}

	p.initWebhookConfig()

	return p, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Config() connector.PluginInternalConfig {
	return p.config
}

func (p *Plugin) Install(ctx context.Context, req connector.InstallRequest) (connector.InstallResponse, error) {
	return connector.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req connector.UninstallRequest) (connector.UninstallResponse, error) {
	if p.client == nil {
		return connector.UninstallResponse{}, connector.ErrNotYetInstalled
	}

	err := p.client.DeleteWebhook(ctx, req.ConnectorID)
	return connector.UninstallResponse{}, err
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return connector.FetchNextAccountsResponse{}, connector.ErrNotYetInstalled
	}
	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req connector.CreateWebhooksRequest) (connector.CreateWebhooksResponse, error) {
	if p.client == nil {
		return connector.CreateWebhooksResponse{}, connector.ErrNotYetInstalled
	}
	configs, err := p.createWebhooks(ctx, req)
	if err != nil {
		return connector.CreateWebhooksResponse{}, err
	}

	others := make([]connector.PSPOther, 0, len(configs))
	for _, config := range configs {
		raw, err := json.Marshal(&config)
		if err != nil {
			return connector.CreateWebhooksResponse{}, err
		}
		others = append(others, connector.PSPOther{
			ID:    config.Name,
			Other: raw,
		})
	}

	return connector.CreateWebhooksResponse{
		Others:  others,
		Configs: configs,
	}, nil
}

func (p *Plugin) VerifyWebhook(ctx context.Context, req connector.VerifyWebhookRequest) (connector.VerifyWebhookResponse, error) {
	if p.client == nil {
		return connector.VerifyWebhookResponse{}, connector.ErrNotYetInstalled
	}

	return p.verifyWebhook(ctx, req)
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req connector.TranslateWebhookRequest) (connector.TranslateWebhookResponse, error) {
	if p.client == nil {
		return connector.TranslateWebhookResponse{}, connector.ErrNotYetInstalled
	}

	config, ok := p.supportedWebhooks[req.Name]
	if !ok {
		return connector.TranslateWebhookResponse{}, errors.New("unknown webhook")
	}

	return config.fn(ctx, req)
}

var _ connector.Plugin = &Plugin{}
