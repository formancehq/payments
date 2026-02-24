package modulr

import (
	"context"
	"encoding/json"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connectors/modulr/client"
	"github.com/formancehq/payments/pkg/registry"
)

const ProviderName = "modulr"

func init() {
	registry.RegisterPlugin(ProviderName, connector.PluginTypePSP, func(_ connector.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (connector.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
    connector.Plugin

    name   string
    logger logging.Logger

    client client.Client
    config Config
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client, err := client.New(ProviderName, config.APIKey, config.APISecret, config.Endpoint)
	if err != nil {
		return nil, err
	}

	return &Plugin{
		Plugin: connector.NewBasePlugin(),

		name:   name,
		logger: logger,
		client: client,
		config: config,
	}, nil
}

func (p *Plugin) Name() string {
    return p.name
}

func (p *Plugin) Config() connector.PluginInternalConfig {
    return p.config
}

func (p *Plugin) Install(ctx context.Context, req connector.InstallRequest) (connector.InstallResponse, error) {
	return connector.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req connector.UninstallRequest) (connector.UninstallResponse, error) {
	return connector.UninstallResponse{}, nil
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	if p.client == nil {
		return connector.FetchNextAccountsResponse{}, connector.ErrNotYetInstalled
	}
	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req connector.FetchNextBalancesRequest) (connector.FetchNextBalancesResponse, error) {
	if p.client == nil {
		return connector.FetchNextBalancesResponse{}, connector.ErrNotYetInstalled
	}
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextExternalAccounts(ctx context.Context, req connector.FetchNextExternalAccountsRequest) (connector.FetchNextExternalAccountsResponse, error) {
	if p.client == nil {
		return connector.FetchNextExternalAccountsResponse{}, connector.ErrNotYetInstalled
	}
	return p.fetchNextExternalAccounts(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req connector.FetchNextPaymentsRequest) (connector.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return connector.FetchNextPaymentsResponse{}, connector.ErrNotYetInstalled
	}
	return p.fetchNextPayments(ctx, req)
}

func (p *Plugin) CreateTransfer(ctx context.Context, req connector.CreateTransferRequest) (connector.CreateTransferResponse, error) {
	if p.client == nil {
		return connector.CreateTransferResponse{}, connector.ErrNotYetInstalled
	}
	payment, err := p.createTransfer(ctx, req.PaymentInitiation)
	if err != nil {
		return connector.CreateTransferResponse{}, err
	}

	return connector.CreateTransferResponse{
		Payment: payment,
	}, nil
}

func (p *Plugin) CreatePayout(ctx context.Context, req connector.CreatePayoutRequest) (connector.CreatePayoutResponse, error) {
	if p.client == nil {
		return connector.CreatePayoutResponse{}, connector.ErrNotYetInstalled
	}
	payment, err := p.createPayout(ctx, req.PaymentInitiation)
	if err != nil {
		return connector.CreatePayoutResponse{}, err
	}
	return connector.CreatePayoutResponse{
		Payment: payment,
	}, nil
}

var _ connector.Plugin = &Plugin{}
