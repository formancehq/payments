package adyen

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/adyen/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "adyen"

func init() {
	registry.RegisterPlugin(ProviderName, func(_ context.Context, _ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{})
}

type Plugin struct {
	models.Plugin

	name   string
	logger logging.Logger

	client            client.Client
	supportedWebhooks map[string]supportedWebhook

	connectorID string
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
		Plugin: plugins.NewBasePlugin(),

		name:   name,
		logger: logger,
		client: client,
	}

	p.initWebhookConfig()

	return p, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	if p.client == nil {
		return models.UninstallResponse{}, plugins.ErrNotYetInstalled
	}

	err := p.client.DeleteWebhook(ctx, req.ConnectorID)
	return models.UninstallResponse{}, err
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return models.FetchNextAccountsResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	if p.client == nil {
		return models.CreateWebhooksResponse{}, plugins.ErrNotYetInstalled
	}
	p.connectorID = req.ConnectorID
	configs, err := p.createWebhooks(ctx, req)
	if err != nil {
		return models.CreateWebhooksResponse{}, err
	}

	others := make([]models.PSPOther, 0, len(configs))
	for _, config := range configs {
		raw, err := json.Marshal(&config)
		if err != nil {
			return models.CreateWebhooksResponse{}, err
		}
		others = append(others, models.PSPOther{
			ID:    config.Name,
			Other: raw,
		})
	}

	return models.CreateWebhooksResponse{
		Others:  others,
		Configs: configs,
	}, nil
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	if p.client == nil {
		return models.TranslateWebhookResponse{}, plugins.ErrNotYetInstalled
	}

	config, ok := p.supportedWebhooks[req.Name]
	if !ok {
		return models.TranslateWebhookResponse{}, errors.New("unknown webhook")
	}

	return config.fn(ctx, req)
}

var _ models.Plugin = &Plugin{}
