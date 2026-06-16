package krakenpro

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/ee/plugins/krakenpro/mappers"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
)

const (
	ProviderName             = "krakenpro"
	MetadataPrefix           = mappers.MetadataPrefix
	assetRefreshTTL          = 24 * time.Hour
	installValidationTimeout = 10 * time.Second
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

// Install validates credentials with a cheap signed call (under a short
// timeout) and warms the asset caches so the first cycle skips the cold
// start. Fatal-auth errors (bad key/secret/permissions) are wrapped as
// models.ErrInvalidRequest to stop Temporal retrying. A transient
// EAPI:Invalid nonce is deliberately NOT fatal (it can be a cross-pod
// race) and is left retriable — see client.fatalAuthCodes / IsRetriableError.
func (p *Plugin) Install(ctx context.Context, _ models.InstallRequest) (models.InstallResponse, error) {
	cctx, cancel := context.WithTimeout(ctx, installValidationTimeout)
	defer cancel()

	if _, err := p.client.GetBalanceEx(cctx); err != nil {
		if client.IsFatalAuthError(err) {
			return models.InstallResponse{}, errors.Wrap(models.ErrInvalidRequest, "install: validate credentials: "+err.Error())
		}
		return models.InstallResponse{}, fmt.Errorf("install: validate credentials: %w", err)
	}
	if err := p.refreshAssets(cctx); err != nil {
		// Asset cache failure is non-fatal at install — the per-read
		// refresh will retry. Log and continue so a flaky asset
		// endpoint can't block the entire install.
		p.logger.WithField("error", err.Error()).
			Errorf("install: asset cache refresh skipped, will retry on demand")
	}

	return models.InstallResponse{Workflow: workflow()}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

// FetchNext* methods are thin guards; inner orchestrators do the work.

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
