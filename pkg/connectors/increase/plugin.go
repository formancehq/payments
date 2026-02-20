package increase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connectors/increase/client"
	"github.com/formancehq/payments/pkg/registry"
)

const ProviderName = "increase"

func init() {
	registry.RegisterPlugin(ProviderName, connector.PluginTypePSP, func(connectorID connector.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (connector.Plugin, error) {
		return New(connectorID, name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
    connector.Plugin

    name   string
    logger logging.Logger

    client              client.Client
    config              Config
    supportedWebhooks   map[client.EventCategory]supportedWebhook
    webhookSharedSecret string
}

func New(connectorID connector.ConnectorID, name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := client.New(ProviderName, config.APIKey, config.Endpoint, config.WebhookSharedSecret)
	p := &Plugin{
		Plugin:              connector.NewBasePlugin(),
		name:                name,
		logger:              logger,
		client:              client,
		webhookSharedSecret: config.WebhookSharedSecret,
		config:              config,
	}

	p.initWebhookConfig()

	return p, nil
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
	if p.client == nil {
		return connector.UninstallResponse{}, connector.ErrNotYetInstalled
	}
	return p.uninstall(ctx, req)
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

func (p *Plugin) FetchNextOthers(ctx context.Context, req connector.FetchNextOthersRequest) (connector.FetchNextOthersResponse, error) {
	return connector.FetchNextOthersResponse{}, connector.ErrNotImplemented
}

func (p *Plugin) CreateBankAccount(ctx context.Context, req connector.CreateBankAccountRequest) (connector.CreateBankAccountResponse, error) {
	if p.client == nil {
		return connector.CreateBankAccountResponse{}, connector.ErrNotYetInstalled
	}
	return p.createBankAccount(ctx, req.BankAccount)
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

func (p *Plugin) ReverseTransfer(ctx context.Context, req connector.ReverseTransferRequest) (connector.ReverseTransferResponse, error) {
	return connector.ReverseTransferResponse{}, connector.ErrNotImplemented
}

func (p *Plugin) PollTransferStatus(ctx context.Context, req connector.PollTransferStatusRequest) (connector.PollTransferStatusResponse, error) {
	return connector.PollTransferStatusResponse{}, connector.ErrNotImplemented
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

func (p *Plugin) ReversePayout(ctx context.Context, req connector.ReversePayoutRequest) (connector.ReversePayoutResponse, error) {
	return connector.ReversePayoutResponse{}, connector.ErrNotImplemented
}

func (p *Plugin) PollPayoutStatus(ctx context.Context, req connector.PollPayoutStatusRequest) (connector.PollPayoutStatusResponse, error) {
	return connector.PollPayoutStatusResponse{}, connector.ErrNotImplemented
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req connector.CreateWebhooksRequest) (connector.CreateWebhooksResponse, error) {
	if p.client == nil {
		return connector.CreateWebhooksResponse{}, connector.ErrNotYetInstalled
	}
	return p.createWebhooks(ctx, req)
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req connector.TranslateWebhookRequest) (connector.TranslateWebhookResponse, error) {
	if p.client == nil {
		return connector.TranslateWebhookResponse{}, connector.ErrNotYetInstalled
	}
	return p.translateWebhook(ctx, req)
}

func (p *Plugin) VerifyWebhook(ctx context.Context, req connector.VerifyWebhookRequest) (connector.VerifyWebhookResponse, error) {
	if p.client == nil {
		return connector.VerifyWebhookResponse{}, connector.ErrNotYetInstalled
	}

	signatures, ok := req.Webhook.Headers[HeadersSignature]
	if !ok || len(signatures) == 0 {
		return connector.VerifyWebhookResponse{}, client.ErrWebhookHeaderXSignatureMissing
	}

	err := p.verifyWebhookSignature(req.Webhook.Body, signatures[0])
	if err != nil {
		return connector.VerifyWebhookResponse{}, fmt.Errorf("%w: %w", err, connector.ErrWebhookVerification)
	}

	if req.Config == nil || req.Config.Name == "" {
		return connector.VerifyWebhookResponse{}, client.ErrWebhookNameUnknown
	}

	if _, ok := p.supportedWebhooks[client.EventCategory(req.Config.Name)]; !ok {
		return connector.VerifyWebhookResponse{}, client.ErrWebhookNameUnknown
	}

	var webhook client.WebhookEvent
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return connector.VerifyWebhookResponse{}, fmt.Errorf("failed to unmarshal webhook: %w", err)
	}

	return connector.VerifyWebhookResponse{
		WebhookIdempotencyKey: &webhook.ID,
	}, nil
}

var _ connector.Plugin = &Plugin{}
