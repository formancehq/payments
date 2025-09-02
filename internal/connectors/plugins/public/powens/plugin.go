package powens

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "powens"

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypeOpenBanking, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{})
}

type Plugin struct {
	models.Plugin

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
		Plugin: plugins.NewBasePlugin(),

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

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	if p.client == nil {
		return models.UninstallResponse{}, plugins.ErrNotYetInstalled
	}

	if err := p.deleteWebhooks(ctx, req); err != nil {
		return models.UninstallResponse{}, err
	}

	return models.UninstallResponse{}, nil
}

func (p *Plugin) CreateUser(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
	if p.client == nil {
		return models.CreateUserResponse{}, plugins.ErrNotYetInstalled
	}

	return p.createUser(ctx, req)
}

func (p *Plugin) CreateUserLink(ctx context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	if p.client == nil {
		return models.CreateUserLinkResponse{}, plugins.ErrNotYetInstalled
	}

	return p.createUserLink(ctx, req)
}

func (p *Plugin) CompleteUserLink(ctx context.Context, req models.CompleteUserLinkRequest) (models.CompleteUserLinkResponse, error) {
	if p.client == nil {
		return models.CompleteUserLinkResponse{}, plugins.ErrNotYetInstalled
	}

	return p.completeUserLink(ctx, req)
}

func (p *Plugin) UpdateUserLink(ctx context.Context, req models.UpdateUserLinkRequest) (models.UpdateUserLinkResponse, error) {
	if p.client == nil {
		return models.UpdateUserLinkResponse{}, plugins.ErrNotYetInstalled
	}

	return p.updateUserLink(ctx, req)
}

func (p *Plugin) DeleteUser(ctx context.Context, req models.DeleteUserRequest) (models.DeleteUserResponse, error) {
	if p.client == nil {
		return models.DeleteUserResponse{}, plugins.ErrNotYetInstalled
	}

	return p.deleteUser(ctx, req)
}

func (p *Plugin) DeleteUserConnection(ctx context.Context, req models.DeleteUserConnectionRequest) (models.DeleteUserConnectionResponse, error) {
	if p.client == nil {
		return models.DeleteUserConnectionResponse{}, plugins.ErrNotYetInstalled
	}

	return p.deleteUserConnection(ctx, req)
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	if p.client == nil {
		return models.CreateWebhooksResponse{}, plugins.ErrNotYetInstalled
	}

	return p.createWebhooks(ctx, req)
}

func (p *Plugin) TrimWebhook(ctx context.Context, req models.TrimWebhookRequest) (models.TrimWebhookResponse, error) {
	if p.client == nil {
		return models.TrimWebhookResponse{}, plugins.ErrNotYetInstalled
	}

	webhookTrimmer, ok := p.supportedWebhooks[client.WebhookEventType(req.Config.Name)]
	if !ok {
		return models.TrimWebhookResponse{}, fmt.Errorf("unsupported webhook event type: %s", req.Config.Name)
	}

	if webhookTrimmer.trimFunction == nil {
		// Nothing to trim, return the webhook as is
		return models.TrimWebhookResponse{
			Webhooks: []models.PSPWebhook{req.Webhook},
		}, nil
	}

	trimmedWebhook, err := webhookTrimmer.trimFunction(ctx, req)
	if err != nil {
		return models.TrimWebhookResponse{}, err
	}

	return trimmedWebhook, nil
}

func (p *Plugin) VerifyWebhook(ctx context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	if p.client == nil {
		return models.VerifyWebhookResponse{}, plugins.ErrNotYetInstalled
	}

	return p.verifyWebhook(ctx, req)
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	if p.client == nil {
		return models.TranslateWebhookResponse{}, plugins.ErrNotYetInstalled
	}

	webhookTranslator, ok := p.supportedWebhooks[client.WebhookEventType(req.Name)]
	if !ok {
		return models.TranslateWebhookResponse{}, fmt.Errorf("unsupported webhook event type: %s", req.Name)
	}

	resp, err := webhookTranslator.handleFunction(ctx, req)
	if err != nil {
		return models.TranslateWebhookResponse{}, err
	}

	return models.TranslateWebhookResponse{
		Responses: resp,
	}, nil
}
