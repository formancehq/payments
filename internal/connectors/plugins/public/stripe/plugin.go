package stripe

import (
	"context"
	"encoding/json"
	stripesdk "github.com/stripe/stripe-go/v79"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "stripe"

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{})
}

type Plugin struct {
	models.Plugin

	name   string
	logger logging.Logger
	client client.Client
	config Config

	supportedWebhooks map[stripesdk.EventType]supportedWebhook
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := client.New(ProviderName, logger, nil, config.APIKey)

	p := &Plugin{
		Plugin: plugins.NewBasePlugin(),

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

func (p *Plugin) Config() models.PluginInternalConfig {
	return p.config
}

func (p *Plugin) Install(_ context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	if p.client == nil {
		return models.UninstallResponse{}, plugins.ErrNotYetInstalled
	}

	// Allows also retrocompatibility
	if len(req.WebhookConfigs) == 0 {
		return models.UninstallResponse{}, nil
	}

	for _, webhookConfig := range req.WebhookConfigs {
		p.logger.Infof("deleting webhook %#v", webhookConfig)
		err := p.client.DeleteWebhook(ctx, webhookConfig.Metadata["webhook_id"])

		if err != nil {
			return models.UninstallResponse{}, err
		}
	}

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

func (p *Plugin) ReverseTransfer(ctx context.Context, req models.ReverseTransferRequest) (models.ReverseTransferResponse, error) {
	if p.client == nil {
		return models.ReverseTransferResponse{}, plugins.ErrNotYetInstalled
	}

	payment, err := p.reverseTransfer(ctx, req.PaymentInitiationReversal)
	if err != nil {
		return models.ReverseTransferResponse{}, err
	}

	return models.ReverseTransferResponse{
		Payment: payment,
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

	return p.createWebhooks(ctx, req)
}

func (p *Plugin) VerifyWebhook(ctx context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	if p.client == nil {
		return models.VerifyWebhookResponse{}, plugins.ErrNotYetInstalled
	}

	return p.verifyWebhook(ctx, req)
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	if p.client == nil {
		return models.TranslateWebhookResponse{}, plugins.ErrNotYetInstalled
	}
	return p.translateWebhook(ctx, req)
}

var _ models.Plugin = &Plugin{}
