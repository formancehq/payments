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

const (
	ProviderName   = "coinbaseprime"
	MetadataPrefix = "com.coinbaseprime.spec/"
)

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypeExchange, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
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
	wallets        map[string]string // symbol → wallet ID (e.g. "USD" → "570270d8-...")
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

	wallets, err := p.loadWalletMap(ctx)
	if err != nil {
		p.logger.Infof("could not load wallet map: %v (wallet IDs will be unavailable on orders)", err)
	}

	p.currencies = currencies
	p.networkSymbols = networkSymbols
	p.wallets = wallets
	p.logger.Infof("loaded %d currencies, %d network aliases, %d wallets from Coinbase Prime", len(currencies), len(networkSymbols), len(wallets))
	return nil
}

func (p *Plugin) refreshWallets(ctx context.Context) error {
	wallets, err := p.loadWalletMap(ctx)
	if err != nil {
		return err
	}
	p.currMu.Lock()
	defer p.currMu.Unlock()
	p.wallets = wallets
	return nil
}

func (p *Plugin) loadWalletMap(ctx context.Context) (map[string]string, error) {
	wallets := make(map[string]string)
	cursor := ""
	for {
		resp, err := p.client.GetWallets(ctx, cursor, 100)
		if err != nil {
			return wallets, fmt.Errorf("listing wallets: %w", err)
		}
		for _, w := range resp.Wallets {
			sym := strings.ToUpper(strings.TrimSpace(w.Symbol))
			if sym != "" {
				wallets[sym] = w.ID
			}
		}
		if !resp.Pagination.HasNext || resp.Pagination.NextCursor == "" {
			break
		}
		cursor = resp.Pagination.NextCursor
	}
	return wallets, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

// ensureCurrencies lazily loads currencies with double-checked locking.
func (p *Plugin) ensureCurrencies(ctx context.Context) error {
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

func (p *Plugin) FetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	if p.client == nil {
		return models.FetchNextOrdersResponse{}, plugins.ErrNotYetInstalled
	}
	if err := p.ensureCurrencies(ctx); err != nil {
		return models.FetchNextOrdersResponse{}, err
	}
	if err := p.refreshWallets(ctx); err != nil {
		p.logger.Infof("could not refresh wallet map: %v", err)
	}
	return p.fetchNextOrders(ctx, req)
}

func (p *Plugin) FetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	if p.client == nil {
		return models.FetchNextConversionsResponse{}, plugins.ErrNotYetInstalled
	}
	if err := p.ensureCurrencies(ctx); err != nil {
		return models.FetchNextConversionsResponse{}, err
	}
	return p.fetchNextConversions(ctx, req)
}

var _ models.Plugin = &Plugin{}
