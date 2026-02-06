package wise

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connectors/wise/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/formancehq/payments/pkg/registry"
)

const ProviderName = "wise"

func init() {
	registry.RegisterPlugin(ProviderName, connector.PluginTypePSP, func(_ connector.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (connector.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{}, PAGE_SIZE)
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
    connector.Plugin

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
		Plugin: connector.NewBasePlugin(),

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

func (p *Plugin) Config() connector.PluginInternalConfig {
    return p.config
}

func (p *Plugin) Install(ctx context.Context, req connector.InstallRequest) (connector.InstallResponse, error) {
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
	return p.fetchExternalAccounts(ctx, req)
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
	case fetchProfileName:
		return p.fetchNextProfiles(ctx, req)
	default:
		return connector.FetchNextOthersResponse{}, connector.ErrNotImplemented
	}
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
	return p.createWebhooks(ctx, req)
}

func (p *Plugin) VerifyWebhook(ctx context.Context, req connector.VerifyWebhookRequest) (connector.VerifyWebhookResponse, error) {
	if p.client == nil {
		return connector.VerifyWebhookResponse{}, connector.ErrNotYetInstalled
	}

	testNotif, ok := req.Webhook.Headers[HeadersTestNotification]
	if ok && len(testNotif) > 0 {
		if testNotif[0] == "true" {
			return connector.VerifyWebhookResponse{}, nil
		}
	}

	v, ok := req.Webhook.Headers[HeadersDeliveryID]
	if !ok || len(v) == 0 {
		return connector.VerifyWebhookResponse{}, fmt.Errorf("%w: %w", ErrWebhookHeaderXDeliveryIDMissing, connector.ErrWebhookVerification)
	}

	signatures, ok := req.Webhook.Headers[HeadersSignature]
	if !ok || len(signatures) == 0 {
		return connector.VerifyWebhookResponse{}, fmt.Errorf("%w: %w", ErrWebhookHeaderXSignatureMissing, connector.ErrWebhookVerification)
	}

	err := p.verifySignature(req.Webhook.Body, signatures[0])
	if err != nil {
		return connector.VerifyWebhookResponse{}, fmt.Errorf("%w: %w", err, connector.ErrWebhookVerification)
	}

	return connector.VerifyWebhookResponse{
		WebhookIdempotencyKey: &v[0],
	}, nil
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req connector.TranslateWebhookRequest) (connector.TranslateWebhookResponse, error) {
	if p.client == nil {
		return connector.TranslateWebhookResponse{}, connector.ErrNotYetInstalled
	}

	testNotif, ok := req.Webhook.Headers[HeadersTestNotification]
	if ok && len(testNotif) > 0 {
		if testNotif[0] == "true" {
			return connector.TranslateWebhookResponse{}, nil
		}
	}

	config, ok := p.supportedWebhooks[req.Name]
	if !ok {
		return connector.TranslateWebhookResponse{}, ErrWebhookNameUnknown
	}

	res, err := config.fn(ctx, req)
	if err != nil {
		return connector.TranslateWebhookResponse{}, err
	}

	return connector.TranslateWebhookResponse{
		Responses: []connector.WebhookResponse{res},
	}, nil
}

func (p *Plugin) SetClient(cl client.Client) {
	p.client = cl
}

var _ connector.Plugin = &Plugin{}
