package adyen

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/adyen/client"
	"github.com/formancehq/payments/internal/models"
)

type Plugin struct {
	client client.Client

	connectorID string
}

func (p *Plugin) Name() string {
	return "adyen"
}

func (p *Plugin) createClient(rawConfig json.RawMessage) error {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return err
	}

	p.client = client.New(
		config.APIKey,
		config.WebhookUsername,
		config.WebhookPassword,
		config.CompanyID,
		config.LiveEndpointPrefix,
	)

	return nil
}

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	if err := p.createClient(req.Config); err != nil {
		return models.InstallResponse{}, err
	}

	p.initWebhookConfig()
	configs := []models.PSPWebhookConfig{}
	for name, c := range webhookConfigs {
		configs = append(configs, models.PSPWebhookConfig{
			Name:    name,
			URLPath: c.urlPath,
		})
	}

	return models.InstallResponse{
		Capabilities:    capabilities,
		Workflow:        workflow(),
		WebhooksConfigs: configs,
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.UninstallResponse{}, err
		}
	}

	err := p.client.DeleteWebhook(ctx, req.ConnectorID)
	return models.UninstallResponse{}, err
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	return models.FetchNextBalancesResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	return models.FetchNextExternalAccountsResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	return models.FetchNextPaymentsResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	return models.FetchNextOthersResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	return models.CreateBankAccountResponse{}, plugins.ErrNotImplemented
}

func (p Plugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	return models.CreateTransferResponse{}, plugins.ErrNotImplemented
}

func (p Plugin) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	return models.PollTransferStatusResponse{}, plugins.ErrNotImplemented
}

func (p Plugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	return models.CreatePayoutResponse{}, plugins.ErrNotImplemented
}

func (p Plugin) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	return models.PollPayoutStatusResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.CreateWebhooksResponse{}, err
		}
	}
	p.connectorID = req.ConnectorID
	err := p.createWebhooks(ctx, req)
	return models.CreateWebhooksResponse{}, err
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.TranslateWebhookResponse{}, err
		}
	}

	config, ok := webhookConfigs[req.Name]
	if !ok {
		return models.TranslateWebhookResponse{}, errors.New("unknown webhook")
	}

	return config.fn(ctx, req)
}

var _ models.Plugin = &Plugin{}
