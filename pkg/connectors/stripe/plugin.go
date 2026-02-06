package stripe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connectors/stripe/client"
	"github.com/formancehq/payments/pkg/registry"
	"github.com/stripe/stripe-go/v80"
)

const ProviderName = "stripe"

func init() {
	registry.RegisterPlugin(ProviderName, connector.PluginTypePSP, func(_ connector.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (connector.Plugin, error) {
		return New(name, logger, rm, nil)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
    connector.Plugin

    name                   string
    logger                 logging.Logger
    client                 client.Client
    config                 Config
}

func New(
	name string,
	logger logging.Logger,
	rawConfig json.RawMessage,
	backend stripe.Backend,
) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client, err := client.New(ProviderName, logger, backend, config.APIKey)
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

func (p *Plugin) Install(_ context.Context, req connector.InstallRequest) (connector.InstallResponse, error) {
	return connector.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req connector.UninstallRequest) (connector.UninstallResponse, error) {
	if p.client == nil {
		return connector.UninstallResponse{}, nil
	}

	if len(req.WebhookConfigs) == 0 {
		return connector.UninstallResponse{}, nil
	}

	err := p.client.DeleteWebhookEndpoints(req.WebhookConfigs)
	if err != nil {
		return connector.UninstallResponse{}, fmt.Errorf("failed to delete stripe webhooks on uninstall: %w", err)
	}
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
		Payment: &payment,
	}, nil
}

func (p *Plugin) ReverseTransfer(ctx context.Context, req connector.ReverseTransferRequest) (connector.ReverseTransferResponse, error) {
	if p.client == nil {
		return connector.ReverseTransferResponse{}, connector.ErrNotYetInstalled
	}

	payment, err := p.reverseTransfer(ctx, req.PaymentInitiationReversal)
	if err != nil {
		return connector.ReverseTransferResponse{}, err
	}

	return connector.ReverseTransferResponse{
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
		Payment: &payment,
	}, nil
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req connector.CreateWebhooksRequest) (connector.CreateWebhooksResponse, error) {
	if p.client == nil {
		return connector.CreateWebhooksResponse{}, connector.ErrNotYetInstalled
	}
	configs, err := p.createWebhooks(ctx, req)
	if err != nil {
		return connector.CreateWebhooksResponse{}, err
	}

	others := make([]connector.PSPOther, 0, len(configs))
	for _, config := range configs {
		raw, err := json.Marshal(&config)
		if err != nil {
			return connector.CreateWebhooksResponse{}, err
		}
		others = append(others, connector.PSPOther{
			ID:    config.Name,
			Other: raw,
		})
	}
	return connector.CreateWebhooksResponse{
		Others:  others,
		Configs: configs,
	}, nil
}

func (p *Plugin) VerifyWebhook(ctx context.Context, req connector.VerifyWebhookRequest) (connector.VerifyWebhookResponse, error) {
	if p.client == nil {
		return connector.VerifyWebhookResponse{}, connector.ErrNotYetInstalled
	}

	idempotencyKey, err := p.verifyWebhook(ctx, req)
	if err != nil {
		return connector.VerifyWebhookResponse{}, err
	}
	return connector.VerifyWebhookResponse{WebhookIdempotencyKey: idempotencyKey}, nil
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req connector.TranslateWebhookRequest) (connector.TranslateWebhookResponse, error) {
	if p.client == nil {
		return connector.TranslateWebhookResponse{}, connector.ErrNotYetInstalled
	}

	responses, err := p.translateWebhook(ctx, req)
	if err != nil {
		return connector.TranslateWebhookResponse{}, err
	}

	return connector.TranslateWebhookResponse{
		Responses: responses,
	}, nil
}

var _ connector.Plugin = &Plugin{}
