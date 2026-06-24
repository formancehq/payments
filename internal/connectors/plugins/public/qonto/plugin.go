package qonto

import (
	"context"
	"encoding/json"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	pkgplugins "github.com/formancehq/payments/pkg/domain/plugins"
	"github.com/formancehq/payments/pkg/domain/models"
)

const ProviderName = "qonto"

var Registration = pkgplugins.Registration{
	PluginType: models.PluginTypePSP,
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

	client client.Client
	config Config
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	clientInstance := client.New(ProviderName, config.ClientID, config.APIKey, config.Endpoint, config.StagingToken)

	return &Plugin{
		Plugin: pkgplugins.NewBasePlugin(),
		name:   name,
		logger: logger,
		client: clientInstance,
		config: config,
	}, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Config() models.PluginInternalConfig {
	return p.config
}

func (p *Plugin) Install(_ context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return models.FetchNextAccountsResponse{}, pkgplugins.ErrNotYetInstalled
	}
	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	if p.client == nil {
		return models.FetchNextBalancesResponse{}, pkgplugins.ErrNotYetInstalled
	}
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	if p.client == nil {
		return models.FetchNextExternalAccountsResponse{}, pkgplugins.ErrNotYetInstalled
	}
	return p.fetchNextExternalAccounts(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return models.FetchNextPaymentsResponse{}, pkgplugins.ErrNotYetInstalled
	}
	return p.fetchNextPayments(ctx, req)
}

var _ models.Plugin = &Plugin{}
