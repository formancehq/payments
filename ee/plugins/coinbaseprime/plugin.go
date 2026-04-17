package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/ee/plugins/coinbaseprime/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const (
	ProviderName        = "coinbaseprime"
	MetadataPrefix      = "com.coinbaseprime.spec/"
	TransactionTypeConversion = "CONVERSION"

	// assetRefreshInterval bounds how often GetPortfolio/GetAssets may be
	// re-fetched. Matches the Fireblocks precedent (ee/plugins/fireblocks).
	assetRefreshInterval = 24 * time.Hour
)

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
	models.Plugin

	name   string
	logger logging.Logger
	client client.Client
	config Config

	// assetsMu protects concurrent reads/writes of the cached reference data
	// below (currencies, networkSymbols, wallets, entityID, assetsLastSync).
	// Reads dominate, so a RWMutex is used.
	assetsMu sync.RWMutex
	// assetsRefreshMu serializes the refresh path so only one goroutine at a
	// time can trigger a GetPortfolio/GetAssets reload. Separate from assetsMu
	// so readers are not blocked during a slow API call.
	assetsRefreshMu sync.Mutex
	assetsLastSync  time.Time

	entityID       string            // portfolio entity ID, fetched once at Install
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
	if err := p.loadAssets(ctx); err != nil {
		return models.InstallResponse{}, fmt.Errorf("loading assets: %w", err)
	}

	wallets, err := p.loadWalletMap(ctx)
	if err != nil {
		// Preserve prior behavior: wallet load failure is non-fatal at Install.
		// Orders will have empty account refs until the next accounts cycle
		// populates p.wallets via fetchNextAccounts.
		p.logger.Infof("could not load wallet map: %v (wallet IDs will be unavailable on orders until next accounts cycle)", err)
	}

	p.assetsMu.Lock()
	p.wallets = wallets
	p.logger.Infof("loaded %d currencies, %d network aliases, %d wallets from Coinbase Prime", len(p.currencies), len(p.networkSymbols), len(p.wallets))
	p.assetsMu.Unlock()

	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

// loadAssets fetches the portfolio entity (once per plugin instance) and the
// asset list, rebuilding the currencies and networkSymbols maps. Callers must
// hold p.assetsRefreshMu to prevent concurrent loads. Stamps assetsLastSync on
// success so ensureAssetsFresh can honor the TTL.
func (p *Plugin) loadAssets(ctx context.Context) error {
	entityID, err := p.ensurePortfolioEntityID(ctx)
	if err != nil {
		return err
	}

	assets, err := p.client.GetAssets(ctx, entityID)
	if err != nil {
		return fmt.Errorf("getting assets for entity %s: %w", entityID, err)
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

	p.assetsMu.Lock()
	p.currencies = currencies
	p.networkSymbols = networkSymbols
	p.assetsLastSync = time.Now()
	p.assetsMu.Unlock()
	return nil
}

// ensurePortfolioEntityID returns the cached portfolio entity ID, fetching it
// from the API once per plugin instance. The entity ID is fixed per API key
// and does not change, so there is no refresh path.
func (p *Plugin) ensurePortfolioEntityID(ctx context.Context) (string, error) {
	p.assetsMu.RLock()
	entityID := p.entityID
	p.assetsMu.RUnlock()
	if entityID != "" {
		return entityID, nil
	}

	portfolio, err := p.client.GetPortfolio(ctx)
	if err != nil {
		return "", fmt.Errorf("getting portfolio: %w", err)
	}

	p.assetsMu.Lock()
	p.entityID = portfolio.Portfolio.EntityID
	entityID = p.entityID
	p.assetsMu.Unlock()
	return entityID, nil
}

// lookupWalletPair resolves a (quoteSymbol, baseSymbol) pair against the
// cached wallet map under a read lock. Returns empty strings for symbols that
// are not mapped.
func (p *Plugin) lookupWalletPair(quoteSymbol, baseSymbol string) (quoteWalletID, baseWalletID string) {
	p.assetsMu.RLock()
	defer p.assetsMu.RUnlock()
	if p.wallets == nil {
		return "", ""
	}
	return p.wallets[strings.ToUpper(quoteSymbol)], p.wallets[strings.ToUpper(baseSymbol)]
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

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return models.FetchNextAccountsResponse{}, plugins.ErrNotYetInstalled
	}
	if err := p.ensureAssetsFresh(ctx); err != nil {
		return models.FetchNextAccountsResponse{}, err
	}
	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	if p.client == nil {
		return models.FetchNextBalancesResponse{}, plugins.ErrNotYetInstalled
	}
	if err := p.ensureAssetsFresh(ctx); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return models.FetchNextPaymentsResponse{}, plugins.ErrNotYetInstalled
	}
	if err := p.ensureAssetsFresh(ctx); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}
	return p.fetchNextPayments(ctx, req)
}

func (p *Plugin) FetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	if p.client == nil {
		return models.FetchNextOrdersResponse{}, plugins.ErrNotYetInstalled
	}
	if err := p.ensureAssetsFresh(ctx); err != nil {
		return models.FetchNextOrdersResponse{}, err
	}
	return p.fetchNextOrders(ctx, req)
}

func (p *Plugin) FetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	if p.client == nil {
		return models.FetchNextConversionsResponse{}, plugins.ErrNotYetInstalled
	}
	if err := p.ensureAssetsFresh(ctx); err != nil {
		return models.FetchNextConversionsResponse{}, err
	}
	return p.fetchNextConversions(ctx, req)
}

var _ models.Plugin = &Plugin{}
