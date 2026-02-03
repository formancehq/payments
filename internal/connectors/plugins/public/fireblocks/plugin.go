package fireblocks

import (
	"context"
	"encoding/json"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/fireblocks/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "fireblocks"

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
	models.Plugin

	name   string
	logger logging.Logger

	client        client.Client
	config        Config
	assetDecimals map[string]int
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	c := client.New(ProviderName, config.APIKey, config.privateKey, config.BaseURL)

	return &Plugin{
		Plugin: plugins.NewBasePlugin(),

		name:   name,
		logger: logger,
		client: c,
		config: config,
	}, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Config() models.PluginInternalConfig {
	return p.config
}

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	assets, err := p.client.ListAssets(ctx)
	if err != nil {
		return models.InstallResponse{}, err
	}

	p.assetDecimals = make(map[string]int, len(assets))
	var skipped int
	for _, asset := range assets {
		if asset.LegacyID == "" {
			skipped++
			continue
		}

		var decimals int
		var hasDecimals bool

		if asset.Onchain != nil {
			decimals = asset.Onchain.Decimals
			hasDecimals = true
		} else if asset.Decimals > 0 {
			decimals = asset.Decimals
			hasDecimals = true
		}

		if !hasDecimals {
			p.logger.Infof("skipping asset %q: no decimals information", asset.LegacyID)
			skipped++
			continue
		}

		if decimals < 0 {
			p.logger.Infof("skipping asset %q: invalid decimals %d", asset.LegacyID, decimals)
			skipped++
			continue
		}

		p.assetDecimals[asset.LegacyID] = decimals
	}

	p.logger.Infof("loaded %d assets from Fireblocks (%d skipped)", len(p.assetDecimals), skipped)

	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
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

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	if p.client == nil {
		return models.FetchNextBalancesResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return models.FetchNextPaymentsResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextPayments(ctx, req)
}

var _ models.Plugin = &Plugin{}
