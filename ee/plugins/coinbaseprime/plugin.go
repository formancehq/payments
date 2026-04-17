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

	// accountLookup is injected by the engine via UseAccountLookup right
	// after plugin instantiation. Orders use it to resolve wallet IDs from
	// the persisted accounts table instead of a fragile in-process cache.
	// Set once and read-only thereafter, so no mutex.
	accountLookup models.AccountLookup

	// assetsMu protects concurrent reads/writes of the cached reference data
	// below (currencies, networkSymbols, entityID, assetsLastSync). Reads
	// dominate, so a RWMutex is used.
	assetsMu sync.RWMutex
	// assetsRefreshMu serializes the refresh path so only one goroutine at a
	// time can trigger a GetPortfolio/GetAssets reload. Separate from assetsMu
	// so readers are not blocked during a slow API call.
	assetsRefreshMu sync.Mutex
	assetsLastSync  time.Time

	entityID       string            // portfolio entity ID, fetched once at Install
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
	if err := p.loadAssets(ctx); err != nil {
		return models.InstallResponse{}, fmt.Errorf("loading assets: %w", err)
	}

	p.assetsMu.RLock()
	p.logger.Infof("loaded %d currencies, %d network aliases from Coinbase Prime", len(p.currencies), len(p.networkSymbols))
	p.assetsMu.RUnlock()

	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

// UseAccountLookup is called by the engine immediately after plugin
// construction to inject a per-connector AccountLookup. Orders use it to
// resolve wallet IDs from the persisted accounts table rather than keeping
// an in-process cache that wouldn't survive pod hops.
func (p *Plugin) UseAccountLookup(lookup models.AccountLookup) {
	p.accountLookup = lookup
}

// BootstrapOnInstall declares that the FetchAccounts task must run to
// completion during the install flow, before any periodic workflows start.
// This guarantees the accounts table is fully populated before FetchOrders
// needs to resolve wallet IDs against it.
func (p *Plugin) BootstrapOnInstall() []models.TaskType {
	return []models.TaskType{models.TASK_FETCH_ACCOUNTS}
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
