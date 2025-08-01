package mangopay

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

const ProviderName = "mangopay"

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{})
}

type Plugin struct {
	models.Plugin

	name   string
	logger logging.Logger

	client            client.Client
	supportedWebhooks map[client.EventType]supportedWebhook
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := client.New(ProviderName, config.ClientID, config.APIKey, config.Endpoint)

	p := &Plugin{
		Plugin: plugins.NewBasePlugin(),

		name:   name,
		logger: logger,
		client: client,
	}

	p.initWebhookConfig()

	return p, nil
}

func (p *Plugin) Name() string {
	return p.name
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

func (p *Plugin) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	if p.client == nil {
		return models.FetchNextExternalAccountsResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextExternalAccounts(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		return models.FetchNextPaymentsResponse{}, plugins.ErrNotYetInstalled
	}
	return p.fetchNextPayments(ctx, req)
}

func (p *Plugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	if p.client == nil {
		return models.FetchNextOthersResponse{}, plugins.ErrNotYetInstalled
	}

	switch req.Name {
	case fetchUsersName:
		return p.fetchNextUsers(ctx, req)
	default:
		return models.FetchNextOthersResponse{}, plugins.ErrNotImplemented
	}
}

func (p *Plugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	if p.client == nil {
		return models.CreateBankAccountResponse{}, plugins.ErrNotYetInstalled
	}
	return p.createBankAccount(ctx, req.BankAccount)
}

func (p *Plugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	if p.client == nil {
		return models.CreateTransferResponse{}, plugins.ErrNotYetInstalled
	}
	payment, err := p.createTransfer(ctx, req.PaymentInitiation)
	if err != nil {
		return models.CreateTransferResponse{}, err
	}

	return models.CreateTransferResponse{
		Payment: &payment,
	}, nil
}

func (p *Plugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	if p.client == nil {
		return models.CreatePayoutResponse{}, plugins.ErrNotYetInstalled
	}

	payment, err := p.createPayout(ctx, req.PaymentInitiation)
	if err != nil {
		return models.CreatePayoutResponse{}, err
	}

	return models.CreatePayoutResponse{
		Payment: &payment,
	}, nil
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	if p.client == nil {
		return models.CreateWebhooksResponse{}, plugins.ErrNotYetInstalled
	}
	configs, err := p.createWebhooks(ctx, req)
	if err != nil {
		return models.CreateWebhooksResponse{}, err
	}

	others := make([]models.PSPOther, 0, len(configs))
	for _, config := range configs {
		raw, err := json.Marshal(&config)
		if err != nil {
			return models.CreateWebhooksResponse{}, err
		}
		others = append(others, models.PSPOther{
			ID:    config.Name,
			Other: raw,
		})
	}
	return models.CreateWebhooksResponse{
		Others:  others,
		Configs: configs,
	}, nil
}

func (p *Plugin) VerifyWebhook(ctx context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	// Nothing to do here, we don't need to verify the webhook and we don't want
	// to generate an idempotency key from the query values
	return models.VerifyWebhookResponse{}, nil
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	if p.client == nil {
		return models.TranslateWebhookResponse{}, plugins.ErrNotYetInstalled
	}

	// Mangopay does not send us the event inside the body, but using
	// URL query.
	eventType, ok := req.Webhook.QueryValues["EventType"]
	if !ok || len(eventType) == 0 {
		return models.TranslateWebhookResponse{}, errorsutils.NewWrappedError(
			fmt.Errorf("missing EventType query parameter"),
			models.ErrInvalidRequest,
		)
	}
	resourceID, ok := req.Webhook.QueryValues["RessourceId"]
	if !ok || len(resourceID) == 0 {
		return models.TranslateWebhookResponse{}, errorsutils.NewWrappedError(
			fmt.Errorf("missing RessourceId query parameter"),
			models.ErrInvalidRequest,
		)
	}
	v, ok := req.Webhook.QueryValues["Date"]
	if !ok || len(v) == 0 {
		return models.TranslateWebhookResponse{}, errorsutils.NewWrappedError(
			fmt.Errorf("missing Date query parameter"),
			models.ErrInvalidRequest,
		)
	}
	date, err := strconv.ParseInt(v[0], 10, 64)
	if err != nil {
		return models.TranslateWebhookResponse{}, errorsutils.NewWrappedError(
			fmt.Errorf("invalid Date query parameter: %w", err),
			models.ErrInvalidRequest,
		)
	}

	webhook := client.Webhook{
		ResourceID: resourceID[0],
		Date:       date,
		EventType:  client.EventType(eventType[0]),
	}

	config, ok := p.supportedWebhooks[webhook.EventType]
	if !ok {
		return models.TranslateWebhookResponse{}, errorsutils.NewWrappedError(
			fmt.Errorf("unsupported webhook event type: %s", webhook.EventType),
			models.ErrInvalidRequest,
		)
	}

	webhookResponse, err := config.fn(ctx, webhookTranslateRequest{
		req:     req,
		webhook: &webhook,
	})
	if err != nil {
		return models.TranslateWebhookResponse{}, err
	}

	return models.TranslateWebhookResponse{
		Responses: []models.WebhookResponse{webhookResponse},
	}, nil
}

var _ models.Plugin = &Plugin{}
