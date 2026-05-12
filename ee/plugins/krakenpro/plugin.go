package krakenpro

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const (
	ProviderName   = "krakenpro"
	MetadataPrefix = "com.krakenpro.spec/"
)

var krakenCurrencies = buildCurrencies()

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
	accountRef string
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

	// Derive a non-secret account reference from the API key
	h := sha256.Sum256([]byte(p.config.APIKey))
	p.accountRef = "kraken-" + hex.EncodeToString(h[:])[:12]
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func buildCurrencies() map[string]int {
	currencies := make(map[string]int, len(fiatCurrenciesFallback)+len(cryptoCurrenciesPrecision))

	maps.Copy(currencies, fiatCurrenciesFallback)
	maps.Copy(currencies, cryptoCurrenciesPrecision)

	return currencies
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

func precisionForAsset(normalizedAsset string) (int, bool) {
	precision, ok := krakenCurrencies[normalizedAsset]
	return precision, ok
}

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

var _ models.Plugin = &Plugin{}
