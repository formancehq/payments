package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/ee/plugins/coinbaseprime/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "coinbaseprime"

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
	models.Plugin

	name           string
	logger         logging.Logger
	client         client.Client
	config         Config
	currMu         sync.Mutex
	currencies     map[string]int    // symbol → decimal precision (loaded dynamically)
	networkSymbols map[string]string // network-scoped symbol → base symbol (e.g. "BASEUSDC" → "USDC")
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	c := client.New(ProviderName, config.APIKey, config.APISecret, config.Passphrase, config.PortfolioID)

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
	if err := p.loadCurrencies(ctx); err != nil {
		return models.InstallResponse{}, fmt.Errorf("loading currencies: %w", err)
	}
	return models.InstallResponse{
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
	networkSymbols := make(map[string]string)

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

		// Build network-scoped symbol aliases (e.g. "BASEUSDC" → "USDC")
		for _, net := range asset.Networks {
			ns := strings.ToUpper(strings.TrimSpace(net.NetworkScopedSymbol))
			if ns != "" && ns != symbol {
				networkSymbols[ns] = symbol
			}
		}
	}

	p.currencies = currencies
	p.networkSymbols = networkSymbols
	p.logger.Infof("loaded %d currencies and %d network aliases from Coinbase Prime", len(currencies), len(networkSymbols))
	return nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

// ensureCurrencies lazily loads currencies with double-checked locking.
func (p *Plugin) ensureCurrencies(ctx context.Context) error {
	if len(p.currencies) > 0 {
		return nil
	}
	p.currMu.Lock()
	defer p.currMu.Unlock()
	if len(p.currencies) > 0 {
		return nil
	}
	return p.loadCurrencies(ctx)
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return models.FetchNextAccountsResponse{}, plugins.ErrNotYetInstalled
	}
	if err := p.ensureCurrencies(ctx); err != nil {
		return models.FetchNextAccountsResponse{}, err
	}
	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	if p.client == nil {
		return models.FetchNextBalancesResponse{}, plugins.ErrNotYetInstalled
	}
	if err := p.ensureCurrencies(ctx); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return models.FetchNextPaymentsResponse{}, plugins.ErrNotYetInstalled
	}
	if err := p.ensureCurrencies(ctx); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}
	return p.fetchNextPayments(ctx, req)
}

var _ models.Plugin = &Plugin{}
