package kraken

import (
	"context"
	"encoding/json"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/kraken/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "kraken"

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
}

func New(
	name string,
	logger logging.Logger,
	rawConfig json.RawMessage,
) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	c := client.New(
		ProviderName,
		config.Endpoint,
		config.PublicKey,
		config.PrivateKey,
	)

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

func (p *Plugin) Install(_ context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
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

func (p *Plugin) FetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	if p.client == nil {
		return models.FetchNextOrdersResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextOrders(ctx, req)
}

func (p *Plugin) CreateOrder(ctx context.Context, req models.CreateOrderRequest) (models.CreateOrderResponse, error) {
	if p.client == nil {
		return models.CreateOrderResponse{}, plugins.ErrNotYetInstalled
	}
	return p.createOrder(ctx, req)
}

func (p *Plugin) CancelOrder(ctx context.Context, req models.CancelOrderRequest) (models.CancelOrderResponse, error) {
	if p.client == nil {
		return models.CancelOrderResponse{}, plugins.ErrNotYetInstalled
	}
	return p.cancelOrder(ctx, req)
}

func (p *Plugin) GetOrderBook(ctx context.Context, req models.GetOrderBookRequest) (models.GetOrderBookResponse, error) {
	if p.client == nil {
		return models.GetOrderBookResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchOrderBook(ctx, req)
}

func (p *Plugin) GetQuote(ctx context.Context, req models.GetQuoteRequest) (models.GetQuoteResponse, error) {
	if p.client == nil {
		return models.GetQuoteResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchQuote(ctx, req)
}

func (p *Plugin) GetTradableAssets(ctx context.Context, req models.GetTradableAssetsRequest) (models.GetTradableAssetsResponse, error) {
	if p.client == nil {
		return models.GetTradableAssetsResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchTradableAssets(ctx, req)
}

func (p *Plugin) GetTicker(ctx context.Context, req models.GetTickerRequest) (models.GetTickerResponse, error) {
	if p.client == nil {
		return models.GetTickerResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchTicker(ctx, req)
}

func (p *Plugin) GetOHLC(ctx context.Context, req models.GetOHLCRequest) (models.GetOHLCResponse, error) {
	if p.client == nil {
		return models.GetOHLCResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchOHLC(ctx, req)
}

func (p *Plugin) FetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	if p.client == nil {
		return models.FetchNextConversionsResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextConversions(ctx, req)
}

func (p *Plugin) CreateConversion(ctx context.Context, req models.CreateConversionRequest) (models.CreateConversionResponse, error) {
	if p.client == nil {
		return models.CreateConversionResponse{}, plugins.ErrNotYetInstalled
	}
	return p.createConversion(ctx, req)
}

var _ models.Plugin = &Plugin{}
