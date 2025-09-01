package dummyopenbanking

import (
	"context"
	"encoding/json"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/dummyopenbanking/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "dummyopenbanking"

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypeBankingBridge, func(connectorID models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, connectorID, rm)
	}, capabilities, Config{})
}

type Plugin struct {
	models.Plugin

	name   string
	config Config
	logger logging.Logger

	client client.Client
}

func New(name string, logger logging.Logger, connectorID models.ConnectorID, rawConfig json.RawMessage) (*Plugin, error) {
	conf, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client, err := client.New(name, conf.Directory, connectorID)
	if err != nil {
		return nil, err
	}

	return &Plugin{
		Plugin: plugins.NewBasePlugin(),

		name:   name,
		logger: logger,
		client: client,
		config: conf,
	}, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return p.install(ctx, req)
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return models.FetchNextAccountsResponse{}, plugins.ErrNotYetInstalled
	}

	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return models.FetchNextPaymentsResponse{}, plugins.ErrNotYetInstalled
	}

	return p.fetchNextPayments(ctx, req)
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
