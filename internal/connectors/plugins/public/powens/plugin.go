package powens

import (
	"context"
	"encoding/json"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "powens"

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypeBankingBridge, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{})
}

type Plugin struct {
	models.Plugin

	name     string
	logger   logging.Logger
	clientID string

	client client.Client
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := client.New(name, config.ClientID, config.ClientSecret, config.Endpoint)

	return &Plugin{
		Plugin: plugins.NewBasePlugin(),

		name:     name,
		logger:   logger,
		clientID: config.ClientID,

		client: client,
	}, nil
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
	return models.UninstallResponse{}, nil
}

func (p *Plugin) CreateUserLink(ctx context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	if p.client == nil {
		return models.CreateUserLinkResponse{}, plugins.ErrNotYetInstalled
	}

	resp, err := p.createUserLink(ctx, req)
	if err != nil {
		return models.CreateUserLinkResponse{}, err
	}

	return resp, nil
}
