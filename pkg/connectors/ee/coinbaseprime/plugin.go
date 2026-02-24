package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connectors/ee/coinbaseprime/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/registry"
)

const ProviderName = "coinbaseprime"

func init() {
	registry.RegisterPlugin(ProviderName, connector.PluginTypePSP, func(_ connector.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (connector.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
	connector.Plugin

	name       string
	logger     logging.Logger
	client     client.Client
	config     Config
	currencies map[string]int // symbol → decimal precision (loaded dynamically)
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	c := client.New(ProviderName, config.APIKey, config.APISecret, config.Passphrase, config.PortfolioID)

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
	if err := p.loadCurrencies(ctx); err != nil {
		return connector.InstallResponse{}, fmt.Errorf("loading currencies: %w", err)
	}
	return connector.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) loadCurrencies(ctx context.Context) error {
	portfolio, err := p.client.GetPortfolio(ctx)
	if err != nil {
		return fmt.Errorf("getting portfolio: %w", err)
	}

	assets, err := p.client.GetAssets(ctx, portfolio.Portfolio.EntityID)
	if err != nil {
		return fmt.Errorf("getting assets for entity %s: %w", portfolio.Portfolio.EntityID, err)
	}

	currencies := make(map[string]int, len(assets.Assets)+len(fiatCurrenciesFallback))

	// Start with fiat fallback
	for k, v := range fiatCurrenciesFallback {
		currencies[k] = v
	}

	// Override/add with dynamic assets from the API
	for _, asset := range assets.Assets {
		symbol := strings.ToUpper(strings.TrimSpace(asset.Symbol))
		if symbol == "" {
			continue
		}

		precision, err := strconv.Atoi(asset.DecimalPrecision)
		if err != nil {
			p.logger.Infof("skipping asset %q: invalid decimal_precision %q", symbol, asset.DecimalPrecision)
			continue
		}

		currencies[symbol] = precision
	}

	p.currencies = currencies
	p.logger.Infof("loaded %d currencies from Coinbase Prime", len(currencies))
	return nil
}

func (p *Plugin) Uninstall(ctx context.Context, req connector.UninstallRequest) (connector.UninstallResponse, error) {
	return connector.UninstallResponse{}, nil
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	if p.client == nil || len(p.currencies) == 0 {
		return connector.FetchNextAccountsResponse{}, connector.ErrNotYetInstalled
	}
	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req connector.FetchNextBalancesRequest) (connector.FetchNextBalancesResponse, error) {
	if p.client == nil || len(p.currencies) == 0 {
		return connector.FetchNextBalancesResponse{}, connector.ErrNotYetInstalled
	}
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req connector.FetchNextPaymentsRequest) (connector.FetchNextPaymentsResponse, error) {
	if p.client == nil || len(p.currencies) == 0 {
		return connector.FetchNextPaymentsResponse{}, connector.ErrNotYetInstalled
	}
	return p.fetchNextPayments(ctx, req)
}

var _ connector.Plugin = &Plugin{}
