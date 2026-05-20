package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const (
	ProviderName = "bitstamp"

	// MetadataPrefix is re-exported from mappers so orchestrator code
	// doesn't import mappers just for the prefix constant.
	MetadataPrefix = mappers.MetadataPrefix

	currencyRefreshInterval = 24 * time.Hour
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

	// Currencies cache: precision map (hot path) + full Currency slice
	// (Networks for enrichment). Both populated by loadCurrencies under
	// a 24h TTL. Double-checked locking on currRefreshMu.
	currMu          sync.RWMutex
	currRefreshMu  sync.Mutex
	currLastSync   time.Time
	currencies     map[string]int // ticker → decimal precision
	currenciesFull []client.Currency

	// derivSkip: process-lifetime set of endpoint paths that returned
	// DerivativesUnsupportedError on this scope. Once flagged, the
	// endpoint is never re-polled — there is no portable way to
	// detect when permissions change.
	derivSkipMu sync.RWMutex
	derivSkip   map[string]struct{}

	// enrichment: install-time caches for markets / my_markets /
	// trading + withdrawal fees. Refreshed in parallel under a 24h TTL.
	enrichment enrichmentState
}

func (p *Plugin) shouldSkipEndpoint(path string) bool {
	p.derivSkipMu.RLock()
	defer p.derivSkipMu.RUnlock()
	_, skip := p.derivSkip[path]
	return skip
}

func (p *Plugin) markEndpointSkipped(path, reason string) {
	p.derivSkipMu.Lock()
	defer p.derivSkipMu.Unlock()
	if _, already := p.derivSkip[path]; already {
		return
	}
	if p.derivSkip == nil {
		p.derivSkip = map[string]struct{}{}
	}
	p.derivSkip[path] = struct{}{}
	p.logger.WithField("endpoint", path).WithField("reason", reason).
		Infof("marking endpoint as not-supported for this account scope; future cycles will skip it")
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = client.DefaultEndpoint
		config.Endpoint = endpoint
	}

	c := client.New(ProviderName, config.APIKey, config.APISecret, endpoint)

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
	// Enrichment failures are non-blocking — the connector continues
	// without the optional metadata. ErrPartialEnrichment is logged
	// inside ensureEnrichment + surfaced for ops visibility but does
	// NOT fail the install.
	if err := p.ensureEnrichment(ctx); err != nil {
		p.logger.WithField("error", err.Error()).
			Errorf("install-time enrichment incomplete; accounts will ship without enrichment metadata")
	}
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) loadCurrencies(ctx context.Context) error {
	currencies, err := p.client.GetCurrencies(ctx)
	if err != nil {
		return fmt.Errorf("failed to get currencies: %w", err)
	}
	currencyMap := make(map[string]int, len(currencies))
	for _, c := range currencies {
		symbol := mappers.NormalizeCurrency(c.Currency)
		if symbol == "" {
			continue
		}
		currencyMap[symbol] = c.Decimals
	}
	p.currMu.Lock()
	p.currencies = currencyMap
	p.currenciesFull = currencies
	p.currLastSync = time.Now()
	p.currMu.Unlock()
	p.logger.Infof("loaded %d currencies from Bitstamp", len(currencyMap))
	return nil
}

// currenciesIndex returns the full Currency objects keyed by
// uppercase ticker. Reads from the shared currency cache populated
// by loadCurrencies — no extra API call.
func (p *Plugin) currenciesIndex(ctx context.Context) (map[string]client.Currency, error) {
	if err := p.ensureCurrencies(ctx); err != nil {
		return nil, err
	}
	p.currMu.RLock()
	defer p.currMu.RUnlock()
	out := make(map[string]client.Currency, len(p.currenciesFull))
	for _, c := range p.currenciesFull {
		out[mappers.NormalizeCurrency(c.Currency)] = c
	}
	return out, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

// ensureCurrencies refreshes Bitstamp currency precision metadata at most once
// per currencyRefreshInterval. Currencies can be added by Bitstamp without a
// connector restart, and payments/balances need fresh precision data.
func (p *Plugin) ensureCurrencies(ctx context.Context) error {
	p.currMu.RLock()
	needsRefresh := len(p.currencies) == 0 || time.Since(p.currLastSync) >= currencyRefreshInterval
	p.currMu.RUnlock()
	if !needsRefresh {
		return nil
	}

	p.currRefreshMu.Lock()
	defer p.currRefreshMu.Unlock()

	p.currMu.RLock()
	needsRefresh = len(p.currencies) == 0 || time.Since(p.currLastSync) >= currencyRefreshInterval
	p.currMu.RUnlock()
	if !needsRefresh {
		return nil
	}
	return p.loadCurrencies(ctx)
}

func (p *Plugin) getCurrencies(ctx context.Context) (map[string]int, error) {
	if err := p.ensureCurrencies(ctx); err != nil {
		return nil, err
	}
	p.currMu.RLock()
	defer p.currMu.RUnlock()
	return p.currencies, nil
}

// FetchNext* methods are thin guards — the inner orchestrators call
// getCurrencies() (which TTL-refreshes under the hood), so freshness
// is handled once at the read site instead of duplicated here.

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

func (p *Plugin) FetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	if p.client == nil {
		return models.FetchNextOrdersResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextOrders(ctx, req)
}

func (p *Plugin) FetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	if p.client == nil {
		return models.FetchNextConversionsResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextConversions(ctx, req)
}

var _ models.Plugin = &Plugin{}
