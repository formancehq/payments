package atlar

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/atlar/client"
	"github.com/formancehq/payments/internal/models"
)

type Plugin struct {
	client client.Client
}

func (p *Plugin) Name() string {
	return "atlar"
}

func (p *Plugin) createClient(rawConfig json.RawMessage) error {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return err
	}

	baseUrl, err := url.Parse(config.BaseURL)
	if err != nil {
		return err
	}

	p.client = client.New(baseUrl, config.AccessKey, config.Secret)

	return nil
}

func (p *Plugin) Install(_ context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	// Check that client can be created
	if err := p.createClient(req.Config); err != nil {
		return models.InstallResponse{}, err
	}

	return models.InstallResponse{
		Capabilities: capabilities,
		Workflow:     workflow(),
	}, nil
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
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
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.FetchNextBalancesResponse{}, err
		}
	}

	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}

	return p.fetchNextExternalAccounts(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	return p.fetchNextPayments(ctx, req)
}

func (p *Plugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	return models.FetchNextOthersResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.CreateBankAccountResponse{}, err
		}
	}

	return p.createBankAccount(ctx, req.BankAccount)
}

func (p *Plugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	return models.CreateTransferResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	return models.PollTransferStatusResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.CreatePayoutResponse{}, err
		}
	}

	payoutID, err := p.createPayout(ctx, req.PaymentInitiation)
	if err != nil {
		return models.CreatePayoutResponse{}, err
	}

	return models.CreatePayoutResponse{
		PollingPayoutID: &payoutID,
	}, nil
}

func (p *Plugin) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.PollPayoutStatusResponse{}, err
		}
	}

	return p.pollPayoutStatus(ctx, req.PayoutID)
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	return models.CreateWebhooksResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	return models.TranslateWebhookResponse{}, plugins.ErrNotImplemented
}

var _ models.Plugin = &Plugin{}
