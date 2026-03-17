package krakenpro

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"sync"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "krakenpro"

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
	models.Plugin

	name       string
	logger     logging.Logger
	client     client.Client
	config     Config
	currOnce         sync.Once
	currencies       map[string]int // normalized asset → decimal precision
	formattedCurrMap map[string]int // pre-computed uppercase map for currency.FormatAsset
	accountRef       string         // derived non-secret account reference
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	var c client.Client
	if config.Endpoint != "" {
		c, err = client.NewWithBaseURL(ProviderName, config.APIKey, config.PrivateKey, config.Endpoint)
	} else {
		c, err = client.New(ProviderName, config.APIKey, config.PrivateKey)
	}
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

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	// Validate credentials by making an authenticated API call
	if _, err := p.client.GetBalance(ctx); err != nil {
		return models.InstallResponse{}, fmt.Errorf("validating credentials: %w", err)
	}

	p.loadCurrencies()
	// Derive a non-secret account reference from the API key
	h := sha256.Sum256([]byte(p.config.APIKey))
	p.accountRef = "kraken-" + hex.EncodeToString(h[:])[:12]
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) loadCurrencies() {
	currencies := make(map[string]int, len(fiatCurrenciesFallback)+len(cryptoCurrenciesPrecision))

	maps.Copy(currencies, fiatCurrenciesFallback)
	maps.Copy(currencies, cryptoCurrenciesPrecision)

	p.currencies = currencies

	// Pre-compute the uppercase currency map for FormatAsset calls
	formatted := make(map[string]int, len(currencies))
	for k, v := range currencies {
		formatted[strings.ToUpper(k)] = v
	}
	p.formattedCurrMap = formatted

	p.logger.Infof("loaded %d currencies for krakenpro", len(currencies))
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

// ensureCurrencies lazily loads currencies using sync.Once for thread safety.
func (p *Plugin) ensureCurrencies() {
	p.currOnce.Do(p.loadCurrencies)
}

// getPrecision returns the decimal precision for a normalized asset code.
// Returns defaultPrecision for unknown assets and logs a warning.
func (p *Plugin) getPrecision(normalizedAsset string) int {
	if precision, ok := p.currencies[normalizedAsset]; ok {
		return precision
	}
	p.logger.Infof("unknown asset %q, using default precision %d", normalizedAsset, defaultPrecision)
	return defaultPrecision
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return models.FetchNextAccountsResponse{}, plugins.ErrNotYetInstalled
	}
	p.ensureCurrencies()
	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	if p.client == nil {
		return models.FetchNextBalancesResponse{}, plugins.ErrNotYetInstalled
	}
	p.ensureCurrencies()
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return models.FetchNextPaymentsResponse{}, plugins.ErrNotYetInstalled
	}
	p.ensureCurrencies()
	return p.fetchNextPayments(ctx, req)
}

var _ models.Plugin = &Plugin{}
