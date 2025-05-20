package wise

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

const ProviderName = "wise"

func init() {
	registry.RegisterPlugin(ProviderName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{})
}

var (
	HeadersTestNotification = "X-Test-Notification"
	HeadersDeliveryID       = "X-Delivery-Id"
	HeadersSignature        = "X-Signature-Sha256"

	ErrStackPublicUrlMissing           = errors.New("STACK_PUBLIC_URL is not set")
	ErrWebhookHeaderXDeliveryIDMissing = errors.New("missing X-Delivery-Id header")
	ErrWebhookHeaderXSignatureMissing  = errors.New("missing X-Signature-Sha256 header")
	ErrWebhookNameUnknown              = errors.New("unknown webhook name")
)

type Plugin struct {
	models.Plugin

	name   string
	logger logging.Logger

	config            Config
	client            client.Client
	supportedWebhooks map[string]supportedWebhook
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := client.New(ProviderName, config.APIKey)

	p := &Plugin{
		Plugin: plugins.NewBasePlugin(),

		name:   name,
		logger: logger,
		client: client,
		config: config,
	}

	p.supportedWebhooks = map[string]supportedWebhook{
		"transfer_state_changed": {
			triggerOn: "transfers#state-change",
			urlPath:   "/transferstatechanged",
			fn:        p.translateTransferStateChangedWebhook,
			version:   "2.0.0",
		},
		"balance_update": {
			triggerOn: "balances#update",
			urlPath:   "/balanceupdate",
			fn:        p.translateBalanceUpdateWebhook,
			version:   "2.2.0",
		},
	}

	return p, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{
		Workflow: workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	if p.client == nil {
		return models.UninstallResponse{}, plugins.ErrNotYetInstalled
	}
	return p.uninstall(ctx, req)
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
	return p.fetchExternalAccounts(ctx, req)
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
	case fetchProfileName:
		return p.fetchNextProfiles(ctx, req)
	default:
		return models.FetchNextOthersResponse{}, plugins.ErrNotImplemented
	}
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
	return p.createWebhooks(ctx, req)
}

func (p *Plugin) VerifyWebhook(ctx context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	if p.client == nil {
		return models.VerifyWebhookResponse{}, plugins.ErrNotYetInstalled
	}

	testNotif, ok := req.Webhook.Headers[HeadersTestNotification]
	if ok && len(testNotif) > 0 {
		if testNotif[0] == "true" {
			return models.VerifyWebhookResponse{}, nil
		}
	}

	v, ok := req.Webhook.Headers[HeadersDeliveryID]
	if !ok || len(v) == 0 {
		return models.VerifyWebhookResponse{}, fmt.Errorf("%w: %w", ErrWebhookHeaderXDeliveryIDMissing, models.ErrWebhookVerification)
	}

	signatures, ok := req.Webhook.Headers[HeadersSignature]
	if !ok || len(signatures) == 0 {
		return models.VerifyWebhookResponse{}, fmt.Errorf("%w: %w", ErrWebhookHeaderXSignatureMissing, models.ErrWebhookVerification)
	}

	err := p.verifySignature(req.Webhook.Body, signatures[0])
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("%w: %w", err, models.ErrWebhookVerification)
	}

	return models.VerifyWebhookResponse{
		WebhookIdempotencyKey: &v[0],
	}, nil
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	if p.client == nil {
		return models.TranslateWebhookResponse{}, plugins.ErrNotYetInstalled
	}

	testNotif, ok := req.Webhook.Headers[HeadersTestNotification]
	if ok && len(testNotif) > 0 {
		if testNotif[0] == "true" {
			return models.TranslateWebhookResponse{}, nil
		}
	}

	config, ok := p.supportedWebhooks[req.Name]
	if !ok {
		return models.TranslateWebhookResponse{}, ErrWebhookNameUnknown
	}

	res, err := config.fn(ctx, req)
	if err != nil {
		return models.TranslateWebhookResponse{}, err
	}

	return models.TranslateWebhookResponse{
		Responses: []models.WebhookResponse{res},
	}, nil
}

func (p *Plugin) SetClient(cl client.Client) {
	p.client = cl
}

var _ models.Plugin = &Plugin{}
