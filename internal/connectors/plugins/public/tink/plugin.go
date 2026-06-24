package tink

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/pkg/domain/models"
	pkgplugins "github.com/formancehq/payments/pkg/domain/plugins"
)

const ProviderName = "tink"

var Registration = pkgplugins.Registration{
	PluginType: models.PluginTypeOpenBanking,
	CreateFunc: func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	},
	Capabilities: capabilities,
	RawConf:      Config{},
	PageSize:     PAGE_SIZE,
}

type Plugin struct {
	models.Plugin

	name   string
	logger logging.Logger

	clientID string
	client   client.Client
	config   Config

	supportedWebhooks map[client.WebhookEventType]supportedWebhook
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := client.New(name, config.ClientID, config.ClientSecret, config.Endpoint)

	p := &Plugin{
		Plugin: pkgplugins.NewBasePlugin(),

		name:   name,
		logger: logger,

		clientID: config.ClientID,
		client:   client,
		config:   config,
	}

	p.initWebhookConfig()

	return p, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Config() models.PluginInternalConfig {
	return p.config
}

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	if p.client == nil {
		return models.UninstallResponse{}, pkgplugins.ErrNotYetInstalled
	}

	err := p.deleteWebhooks(ctx, req)
	if err != nil {
		return models.UninstallResponse{}, err
	}

	return models.UninstallResponse{}, nil
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return models.FetchNextAccountsResponse{}, pkgplugins.ErrNotYetInstalled
	}

	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return models.FetchNextPaymentsResponse{}, pkgplugins.ErrNotYetInstalled
	}
	return p.fetchNextPayments(ctx, req)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	if p.client == nil {
		return models.FetchNextBalancesResponse{}, pkgplugins.ErrNotYetInstalled
	}
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) CreateUser(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
	if p.client == nil {
		return models.CreateUserResponse{}, pkgplugins.ErrNotYetInstalled
	}

	return p.createUser(ctx, req)
}

func (p *Plugin) CreateUserLink(ctx context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	if p.client == nil {
		return models.CreateUserLinkResponse{}, pkgplugins.ErrNotYetInstalled
	}

	return p.createUserLink(ctx, req)
}

func (p *Plugin) UpdateUserLink(ctx context.Context, req models.UpdateUserLinkRequest) (models.UpdateUserLinkResponse, error) {
	if p.client == nil {
		return models.UpdateUserLinkResponse{}, pkgplugins.ErrNotYetInstalled
	}

	return p.updateUserLink(ctx, req)
}

func (p *Plugin) CompleteUserLink(ctx context.Context, req models.CompleteUserLinkRequest) (models.CompleteUserLinkResponse, error) {
	if p.client == nil {
		return models.CompleteUserLinkResponse{}, pkgplugins.ErrNotYetInstalled
	}

	return p.completeUserLink(ctx, req)
}

func (p *Plugin) DeleteUserConnection(ctx context.Context, req models.DeleteUserConnectionRequest) (models.DeleteUserConnectionResponse, error) {
	if p.client == nil {
		return models.DeleteUserConnectionResponse{}, pkgplugins.ErrNotYetInstalled
	}

	return p.deleteUserConnection(ctx, req)
}

func (p *Plugin) DeleteUser(ctx context.Context, req models.DeleteUserRequest) (models.DeleteUserResponse, error) {
	if p.client == nil {
		return models.DeleteUserResponse{}, pkgplugins.ErrNotYetInstalled
	}

	return p.deleteUser(ctx, req)
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	if p.client == nil {
		return models.CreateWebhooksResponse{}, pkgplugins.ErrNotYetInstalled
	}

	return p.createWebhooks(ctx, req)
}

func (p *Plugin) VerifyWebhook(ctx context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	if p.client == nil {
		return models.VerifyWebhookResponse{}, pkgplugins.ErrNotYetInstalled
	}

	return p.verifyWebhook(ctx, req)
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	if p.client == nil {
		return models.TranslateWebhookResponse{}, pkgplugins.ErrNotYetInstalled
	}

	if req.Name == "" {
		return models.TranslateWebhookResponse{}, fmt.Errorf("missing webhook name: %w", models.ErrInvalidRequest)
	}

	webhookTranslator, ok := p.supportedWebhooks[client.WebhookEventType(req.Name)]
	if !ok {
		return models.TranslateWebhookResponse{}, fmt.Errorf("unsupported webhook event type: %s", req.Name)
	}

	resp, err := webhookTranslator.fn(ctx, req)
	if err != nil {
		return models.TranslateWebhookResponse{}, err
	}

	return models.TranslateWebhookResponse{
		Responses: resp,
	}, nil
}
