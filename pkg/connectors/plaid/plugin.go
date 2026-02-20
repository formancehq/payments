package plaid

import (
	"context"
	"encoding/json"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connectors/plaid/client"
	"github.com/formancehq/payments/pkg/registry"
)

const ProviderName = "plaid"

func init() {
	registry.RegisterPlugin(ProviderName, connector.PluginTypeOpenBanking, func(connectorID connector.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (connector.Plugin, error) {
		return New(name, logger, connectorID, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
    connector.Plugin

    name   string
    logger logging.Logger

    client client.Client
    config Config

    supportedWebhooks map[string]supportedWebhook

}

func New(name string, logger logging.Logger, connectorID connector.ConnectorID, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client, err := client.New(name, config.ClientID, config.ClientSecret, connectorID, config.IsSandbox)
	if err != nil {
		return nil, err
	}

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
	return connector.UninstallResponse{}, nil
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return connector.FetchNextAccountsResponse{}, connector.ErrNotYetInstalled
	}

	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req connector.FetchNextBalancesRequest) (connector.FetchNextBalancesResponse, error) {
	if p.client == nil {
		return connector.FetchNextBalancesResponse{}, connector.ErrNotYetInstalled
	}
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req connector.FetchNextPaymentsRequest) (connector.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return connector.FetchNextPaymentsResponse{}, connector.ErrNotYetInstalled
	}

	return p.fetchNextPayments(ctx, req)
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

	return p.translateWebhook(ctx, req)
}
