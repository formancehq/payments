package krakenpro

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
)

const (
	ProviderName    = "krakenpro"
	assetRefreshTTL = 24 * time.Hour
)

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

// Plugin is the Kraken Pro EE connector instance. The asset caches
// (currencies + pairs) live in-process with a 24h TTL refresh: the
// engine never injects a persistent cache, so each connector instance
// rebuilds them on first use and refreshes them on demand.
type Plugin struct {
	models.Plugin

	name   string
	logger logging.Logger
	client client.Client
	config Config

	// Asset / pair caches. Reads dominate writes, so an RWMutex
	// protects the published maps; a separate Mutex serialises
	// the refresh path so two callers can't race a duplicate
	// upstream call.
	assetsMu      sync.RWMutex
	assetsRefresh sync.Mutex
	assetsLoaded  time.Time
	currencies    map[string]int // canonical symbol → precision
	assetPairs    map[string]client.AssetPair
	// assetCodes maps a canonical symbol → its raw Kraken spot code
	// (the suffix-free /0/public/Assets key, e.g. BTC → XXBT). It is
	// the deterministic source for the spot/trading account reference
	// even when BalanceEx returns only an earn variant (XBT.M).
	assetCodes map[string]string
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	c, err := client.New(ProviderName, config.APIKey, config.APISecret, config.Endpoint)
	if err != nil {
		return nil, err
	}

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

// Install only registers the periodic workflow — it does no network I/O,
// so install stays fast. Validation is deferred: New already rejects
// malformed config, the asset caches lazy-load on first fetch (see
// ensureAssets), and a bad-but-well-formed key surfaces on the first
// poll as a fatal-auth error mapped to a non-retryable error
// (mapFatalAuth), the same class Install used to catch.
func (p *Plugin) Install(_ context.Context, _ models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{Workflow: workflow()}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

// FetchNext* methods are thin guards; inner orchestrators do the work.
// Each maps a fatal Kraken auth failure to a non-retryable error so a
// revoked key / permission change stops the activity instead of looping.

// mapFatalAuth converts a fatal Kraken auth failure into a non-retryable
// models.ErrInvalidRequest (the registry only short-circuits retries on
// that), mirroring how Install treated the same error class. Everything
// else — including retriable rate-limit/nonce — passes through unchanged.
func mapFatalAuth(err error) error {
	if err != nil && client.IsFatalAuthError(err) {
		return errors.Wrap(models.ErrInvalidRequest, err.Error())
	}
	return err
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return models.FetchNextAccountsResponse{}, plugins.ErrNotYetInstalled
	}
	resp, err := p.fetchNextAccounts(ctx, req)
	return resp, mapFatalAuth(err)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	if p.client == nil {
		return models.FetchNextBalancesResponse{}, plugins.ErrNotYetInstalled
	}
	resp, err := p.fetchNextBalances(ctx, req)
	return resp, mapFatalAuth(err)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return models.FetchNextPaymentsResponse{}, plugins.ErrNotYetInstalled
	}
	resp, err := p.fetchNextPayments(ctx, req)
	return resp, mapFatalAuth(err)
}

func (p *Plugin) FetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	if p.client == nil {
		return models.FetchNextOrdersResponse{}, plugins.ErrNotYetInstalled
	}
	resp, err := p.fetchNextOrders(ctx, req)
	return resp, mapFatalAuth(err)
}

func (p *Plugin) FetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	if p.client == nil {
		return models.FetchNextConversionsResponse{}, plugins.ErrNotYetInstalled
	}
	resp, err := p.fetchNextConversions(ctx, req)
	return resp, mapFatalAuth(err)
}

// logCycle writes the standard end-of-cycle log line every
// orchestrator emits. Kept here so every fetch_* task uses the same
// field names — downstream log queries / dashboards can rely on a
// stable schema. `extras` is a variadic key/value list (key first,
// value second, alternating) for task-specific fields like
// fetch_conversions' `pending` counter.
func (p *Plugin) logCycle(
	task string,
	emitted, pageRows int,
	w ledgerWindow,
	hasMore bool,
	extras ...any,
) {
	l := p.logger.
		WithField("emitted", emitted).
		WithField("rowsThisPage", pageRows).
		WithField("draining", w.draining()).
		WithField("offset", w.Offset).
		WithField("hasMore", hasMore)
	for i := 0; i+1 < len(extras); i += 2 {
		key, ok := extras[i].(string)
		if !ok {
			continue
		}
		l = l.WithField(key, extras[i+1])
	}
	l.Infof("krakenpro %s cycle done", task)
}

var _ models.Plugin = &Plugin{}
