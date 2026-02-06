package fireblocks

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connectors/fireblocks/client"
	"github.com/formancehq/payments/pkg/registry"
)

const ProviderName = "fireblocks"
const assetRefreshInterval = 24 * time.Hour

func init() {
	registry.RegisterPlugin(ProviderName, connector.PluginTypePSP, func(_ connector.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (connector.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
	connector.Plugin

	name   string
	logger logging.Logger

	client client.Client
	config Config

	assetsMu        sync.RWMutex
	assetsRefreshMu sync.Mutex
	assetsLastSync  time.Time
	assetDecimals   map[string]int
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	c := client.New(ProviderName, config.APIKey, config.privateKey, config.Endpoint)

	return &Plugin{
		Plugin: connector.NewBasePlugin(),

		name:   name,
		logger: logger,
		client: c,
		config: config,
	}, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Config() connector.PluginInternalConfig {
	return p.config
}

func (p *Plugin) Install(ctx context.Context, req connector.InstallRequest) (connector.InstallResponse, error) {
	if err := p.loadAssets(ctx); err != nil {
		return connector.InstallResponse{}, err
	}

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
	if err := p.ensureAssetsFresh(ctx); err != nil {
		return connector.FetchNextBalancesResponse{}, err
	}
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req connector.FetchNextPaymentsRequest) (connector.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return connector.FetchNextPaymentsResponse{}, connector.ErrNotYetInstalled
	}
	if err := p.ensureAssetsFresh(ctx); err != nil {
		return connector.FetchNextPaymentsResponse{}, err
	}
	return p.fetchNextPayments(ctx, req)
}

var _ connector.Plugin = &Plugin{}
