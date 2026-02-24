package mangopay

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/connectors/mangopay/client"
	"github.com/formancehq/payments/pkg/registry"
)

const ProviderName = "mangopay"

func init() {
	registry.RegisterPlugin(ProviderName, connector.PluginTypePSP, func(_ connector.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (connector.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
}

type Plugin struct {
    connector.Plugin

    name   string
    logger logging.Logger

    client            client.Client
    config            Config
    supportedWebhooks map[client.EventType]supportedWebhook
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := client.New(ProviderName, config.ClientID, config.APIKey, config.Endpoint)

	p := &Plugin{
		Plugin: connector.NewBasePlugin(),

		name:   name,
		logger: logger,
		client: client,
		config: config,
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

func (p *Plugin) FetchNextOthers(ctx context.Context, req connector.FetchNextOthersRequest) (connector.FetchNextOthersResponse, error) {
	if p.client == nil {
		return connector.FetchNextOthersResponse{}, connector.ErrNotYetInstalled
	}

	switch req.Name {
	case fetchUsersName:
		return p.fetchNextUsers(ctx, req)
	default:
		return connector.FetchNextOthersResponse{}, connector.ErrNotImplemented
	}
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
		Payment: &payment,
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
	// Nothing to do here, we don't need to verify the webhook and we don't want
	// to generate an idempotency key from the query values
	return connector.VerifyWebhookResponse{}, nil
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req connector.TranslateWebhookRequest) (connector.TranslateWebhookResponse, error) {
	if p.client == nil {
		return connector.TranslateWebhookResponse{}, connector.ErrNotYetInstalled
	}

	// Mangopay does not send us the event inside the body, but using
	// URL query.
	eventType, ok := req.Webhook.QueryValues["EventType"]
	if !ok || len(eventType) == 0 {
		return connector.TranslateWebhookResponse{}, connector.NewWrappedError(
			fmt.Errorf("missing EventType query parameter"),
			connector.ErrInvalidRequest,
		)
	}
	resourceID, ok := req.Webhook.QueryValues["RessourceId"]
	if !ok || len(resourceID) == 0 {
		return connector.TranslateWebhookResponse{}, connector.NewWrappedError(
			fmt.Errorf("missing RessourceId query parameter"),
			connector.ErrInvalidRequest,
		)
	}
	v, ok := req.Webhook.QueryValues["Date"]
	if !ok || len(v) == 0 {
		return connector.TranslateWebhookResponse{}, connector.NewWrappedError(
			fmt.Errorf("missing Date query parameter"),
			connector.ErrInvalidRequest,
		)
	}
	date, err := strconv.ParseInt(v[0], 10, 64)
	if err != nil {
		return connector.TranslateWebhookResponse{}, connector.NewWrappedError(
			fmt.Errorf("invalid Date query parameter: %w", err),
			connector.ErrInvalidRequest,
		)
	}

	webhook := client.Webhook{
		ResourceID: resourceID[0],
		Date:       date,
		EventType:  client.EventType(eventType[0]),
	}

	config, ok := p.supportedWebhooks[webhook.EventType]
	if !ok {
		return connector.TranslateWebhookResponse{}, connector.NewWrappedError(
			fmt.Errorf("unsupported webhook event type: %s", webhook.EventType),
			connector.ErrInvalidRequest,
		)
	}

	webhookResponse, err := config.fn(ctx, webhookTranslateRequest{
		req:     req,
		webhook: &webhook,
	})
	if err != nil {
		return connector.TranslateWebhookResponse{}, err
	}

	return connector.TranslateWebhookResponse{
		Responses: []connector.WebhookResponse{webhookResponse},
	}, nil
}

var _ connector.Plugin = &Plugin{}
