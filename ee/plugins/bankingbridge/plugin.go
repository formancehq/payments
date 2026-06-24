package bankingbridge

import (
	"context"
	"encoding/json"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/bankingbridge/client"
	"github.com/formancehq/payments/pkg/domain/models"
	pkgplugins "github.com/formancehq/payments/pkg/domain/plugins"
)

const (
	ProviderName = "bankingbridge"

	MetadataPrefix = "cloud.formance.banking-bridge.spec/"
)

var Registration = pkgplugins.Registration{
	PluginType: models.PluginTypePSP,
	CreateFunc: func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	},
	Capabilities: capabilities,
	RawConf:      Config{},
	PageSize:     PAGE_SIZE,
}

type Plugin struct {
	models.Plugin

	name   string
	logger logging.Logger

	config Config
	client client.Client
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := client.New(name, config.ClientID, config.ClientSecret, config.AuthEndpoint, config.Endpoint)
	return &Plugin{
		Plugin: pkgplugins.NewBasePlugin(),

		name:   name,
		logger: logger,
		config: config,
		client: client,
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
		return models.FetchNextAccountsResponse{}, pkgplugins.ErrNotYetInstalled
	}
	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	if p.client == nil {
		return models.FetchNextBalancesResponse{}, pkgplugins.ErrNotYetInstalled
	}
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	return models.FetchNextExternalAccountsResponse{}, pkgplugins.ErrNotImplemented
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return models.FetchNextPaymentsResponse{}, pkgplugins.ErrNotYetInstalled
	}
	return p.fetchNextPayments(ctx, req)
}

func (p *Plugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	return models.FetchNextOthersResponse{}, pkgplugins.ErrNotImplemented
}

func (p *Plugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	return models.CreateBankAccountResponse{}, pkgplugins.ErrNotImplemented
}

func (p *Plugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	return models.CreateTransferResponse{}, pkgplugins.ErrNotImplemented
}

func (p *Plugin) ReverseTransfer(ctx context.Context, req models.ReverseTransferRequest) (models.ReverseTransferResponse, error) {
	return models.ReverseTransferResponse{}, pkgplugins.ErrNotImplemented
}

func (p *Plugin) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	return models.PollTransferStatusResponse{}, pkgplugins.ErrNotImplemented
}

func (p *Plugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	return models.CreatePayoutResponse{}, pkgplugins.ErrNotImplemented
}

func (p *Plugin) ReversePayout(ctx context.Context, req models.ReversePayoutRequest) (models.ReversePayoutResponse, error) {
	return models.ReversePayoutResponse{}, pkgplugins.ErrNotImplemented
}

func (p *Plugin) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	return models.PollPayoutStatusResponse{}, pkgplugins.ErrNotImplemented
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	return models.CreateWebhooksResponse{}, pkgplugins.ErrNotImplemented
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	return models.TranslateWebhookResponse{}, pkgplugins.ErrNotImplemented
}

var _ models.Plugin = &Plugin{}
