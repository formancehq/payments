package atlar

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connectors/atlar/client"
	"github.com/formancehq/payments/pkg/registry"
)

const ProviderName = "atlar"

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

	baseUrl, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, err
	}

	return &Plugin{
		Plugin: connector.NewBasePlugin(),

		name:   name,
		logger: logger,
		client: client.New(ProviderName, baseUrl, config.AccessKey, config.Secret),
		config: config,
	}, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Config() connector.PluginInternalConfig {
	return p.config
}

func (p *Plugin) Install(_ context.Context, req connector.InstallRequest) (connector.InstallResponse, error) {
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

func (p *Plugin) CreateBankAccount(ctx context.Context, req connector.CreateBankAccountRequest) (connector.CreateBankAccountResponse, error) {
	if p.client == nil {
		return connector.CreateBankAccountResponse{}, connector.ErrNotYetInstalled
	}
	return p.createBankAccount(ctx, req.BankAccount)
}

func (p *Plugin) CreatePayout(ctx context.Context, req connector.CreatePayoutRequest) (connector.CreatePayoutResponse, error) {
	if p.client == nil {
		return connector.CreatePayoutResponse{}, connector.ErrNotYetInstalled
	}

	payoutID, err := p.createPayout(ctx, req.PaymentInitiation)
	if err != nil {
		return connector.CreatePayoutResponse{}, err
	}

	return connector.CreatePayoutResponse{
		PollingPayoutID: &payoutID,
	}, nil
}

func (p *Plugin) PollPayoutStatus(ctx context.Context, req connector.PollPayoutStatusRequest) (connector.PollPayoutStatusResponse, error) {
	if p.client == nil {
		return connector.PollPayoutStatusResponse{}, connector.ErrNotYetInstalled
	}

	return p.pollPayoutStatus(ctx, req.PayoutID)
}

var _ connector.Plugin = &Plugin{}
