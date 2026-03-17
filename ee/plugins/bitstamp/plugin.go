package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "bitstamp"

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
	currLoaded     atomic.Bool
	currencies     map[string]int // currency ticker (uppercase) → decimal precision
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	var c client.Client
	if config.BaseURL != "" {
		c = client.NewWithBaseURL(ProviderName, config.APIKey, config.APISecret, config.BaseURL)
	} else {
		c = client.New(ProviderName, config.APIKey, config.APISecret)
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

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	if err := p.loadCurrencies(ctx); err != nil {
		return models.InstallResponse{}, fmt.Errorf("loading currencies: %w", err)
	}
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) loadCurrencies(ctx context.Context) error {
	currencies, err := p.client.GetCurrencies(ctx)
	if err != nil {
		return fmt.Errorf("getting currencies: %w", err)
	}

	currencyMap := make(map[string]int, len(currencies))
	for _, c := range currencies {
		symbol := normalizeCurrency(c.Currency)
		if symbol == "" {
			continue
		}
		currencyMap[symbol] = c.Decimals
	}

	p.currencies = currencyMap
	p.currLoaded.Store(true)
	p.logger.Infof("loaded %d currencies from Bitstamp", len(currencyMap))
	return nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

// ensureCurrencies lazily loads currencies using an atomic flag for the fast
// path and a mutex for the slow path, avoiding data races on the map read.
func (p *Plugin) ensureCurrencies(ctx context.Context) error {
	if p.currLoaded.Load() {
		return nil
	}
	p.currMu.Lock()
	defer p.currMu.Unlock()
	if p.currLoaded.Load() {
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
