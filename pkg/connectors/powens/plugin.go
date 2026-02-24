package powens

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connectors/powens/client"
	"github.com/formancehq/payments/pkg/registry"
)

const ProviderName = "powens"

func init() {
	registry.RegisterPlugin(ProviderName, connector.PluginTypeOpenBanking, func(_ connector.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (connector.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
    connector.Plugin

    name     string
    logger   logging.Logger
    clientID string

    client client.Client
    config Config

    supportedWebhooks map[client.WebhookEventType]supportedWebhook
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client, err := client.New(name, config.ClientID, config.ClientSecret, config.ConfigurationToken, config.Endpoint)
	if err != nil {
		return nil, err
	}

	p := &Plugin{
		Plugin: connector.NewBasePlugin(),

		name:     name,
		logger:   logger,
		clientID: config.ClientID,

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

	if err := p.deleteWebhooks(ctx, req); err != nil {
		return connector.UninstallResponse{}, err
	}

	return connector.UninstallResponse{}, nil
}

func (p *Plugin) CreateUser(ctx context.Context, req connector.CreateUserRequest) (connector.CreateUserResponse, error) {
	if p.client == nil {
		return connector.CreateUserResponse{}, connector.ErrNotYetInstalled
	}

	return p.createUser(ctx, req)
}

func (p *Plugin) CreateUserLink(ctx context.Context, req connector.CreateUserLinkRequest) (connector.CreateUserLinkResponse, error) {
	if p.client == nil {
		return connector.CreateUserLinkResponse{}, connector.ErrNotYetInstalled
	}

	return p.createUserLink(ctx, req)
}

func (p *Plugin) CompleteUserLink(ctx context.Context, req connector.CompleteUserLinkRequest) (connector.CompleteUserLinkResponse, error) {
	if p.client == nil {
		return connector.CompleteUserLinkResponse{}, connector.ErrNotYetInstalled
	}

	return p.completeUserLink(ctx, req)
}

func (p *Plugin) UpdateUserLink(ctx context.Context, req connector.UpdateUserLinkRequest) (connector.UpdateUserLinkResponse, error) {
	if p.client == nil {
		return connector.UpdateUserLinkResponse{}, connector.ErrNotYetInstalled
	}

	return p.updateUserLink(ctx, req)
}

func (p *Plugin) DeleteUser(ctx context.Context, req connector.DeleteUserRequest) (connector.DeleteUserResponse, error) {
	if p.client == nil {
		return connector.DeleteUserResponse{}, connector.ErrNotYetInstalled
	}

	return p.deleteUser(ctx, req)
}

func (p *Plugin) DeleteUserConnection(ctx context.Context, req connector.DeleteUserConnectionRequest) (connector.DeleteUserConnectionResponse, error) {
	if p.client == nil {
		return connector.DeleteUserConnectionResponse{}, connector.ErrNotYetInstalled
	}

	return p.deleteUserConnection(ctx, req)
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req connector.CreateWebhooksRequest) (connector.CreateWebhooksResponse, error) {
	if p.client == nil {
		return connector.CreateWebhooksResponse{}, connector.ErrNotYetInstalled
	}

	return p.createWebhooks(ctx, req)
}

func (p *Plugin) TrimWebhook(ctx context.Context, req connector.TrimWebhookRequest) (connector.TrimWebhookResponse, error) {
	if p.client == nil {
		return connector.TrimWebhookResponse{}, connector.ErrNotYetInstalled
	}

	webhookTrimmer, ok := p.supportedWebhooks[client.WebhookEventType(req.Config.Name)]
	if !ok {
		return connector.TrimWebhookResponse{}, fmt.Errorf("unsupported webhook event type: %s", req.Config.Name)
	}

	if webhookTrimmer.trimFunction == nil {
		// Nothing to trim, return the webhook as is
		return connector.TrimWebhookResponse{
			Webhooks: []connector.PSPWebhook{req.Webhook},
		}, nil
	}

	trimmedWebhook, err := webhookTrimmer.trimFunction(ctx, req)
	if err != nil {
		return connector.TrimWebhookResponse{}, err
	}

	return trimmedWebhook, nil
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

	webhookTranslator, ok := p.supportedWebhooks[client.WebhookEventType(req.Name)]
	if !ok {
		return connector.TranslateWebhookResponse{}, fmt.Errorf("unsupported webhook event type: %s", req.Name)
	}

	resp, err := webhookTranslator.handleFunction(ctx, req)
	if err != nil {
		return connector.TranslateWebhookResponse{}, err
	}

	return connector.TranslateWebhookResponse{
		Responses: resp,
	}, nil
}
