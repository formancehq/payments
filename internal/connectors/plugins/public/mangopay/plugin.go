package mangopay

import (
	"context"
	"fmt"
	"strconv"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
)

type Plugin struct {
	client client.Client
}

func (p *Plugin) Name() string {
	return "mangopay"
}

func (p *Plugin) createClient(rawConfig []byte) error {
	config, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return err
	}

	p.client = client.New(config.ClientID, config.APIKey, config.Endpoint)

	return nil
}

func (p *Plugin) Install(_ context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	if err := p.createClient(req.Config); err != nil {
		return models.InstallResponse{}, err
	}

	p.initWebhookConfig()

	configs := make([]models.PSPWebhookConfig, 0, len(webhookConfigs))
	for name, config := range webhookConfigs {
		configs = append(configs, models.PSPWebhookConfig{
			Name:    string(name),
			URLPath: config.urlPath,
		})
	}

	return models.InstallResponse{
		Capabilities:    capabilities,
		WebhooksConfigs: configs,
		Workflow:        workflow(),
	}, nil
}

func (p Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

func (p Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	return p.fetchNextAccounts(ctx, req)
}

func (p Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.FetchNextBalancesResponse{}, err
		}
	}

	return p.fetchNextBalances(ctx, req)
}

func (p Plugin) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}

	return p.fetchNextExternalAccounts(ctx, req)
}

func (p Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	return p.fetchNextPayments(ctx, req)
}

func (p Plugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.FetchNextOthersResponse{}, err
		}
	}

	switch req.Name {
	case fetchUsersName:
		return p.fetchNextUsers(ctx, req)
	default:
		return models.FetchNextOthersResponse{}, plugins.ErrNotImplemented
	}
}

func (p Plugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.CreateBankAccountResponse{}, err
		}
	}

	return p.createBankAccount(ctx, req.BankAccount)
}

func (p Plugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.CreateTransferResponse{}, err
		}
	}

	payment, err := p.createTransfer(ctx, req.PaymentInitiation)
	if err != nil {
		return models.CreateTransferResponse{}, err
	}

	return models.CreateTransferResponse{
		Payment: &payment,
	}, nil
}

func (p Plugin) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	return models.PollTransferStatusResponse{}, plugins.ErrNotImplemented
}

func (p Plugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.CreatePayoutResponse{}, err
		}
	}

	payment, err := p.createPayout(ctx, req.PaymentInitiation)
	if err != nil {
		return models.CreatePayoutResponse{}, err
	}

	return models.CreatePayoutResponse{
		Payment: &payment,
	}, nil
}

func (p Plugin) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	return models.PollPayoutStatusResponse{}, plugins.ErrNotImplemented
}

func (p Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.CreateWebhooksResponse{}, err
		}
	}

	err := p.createWebhooks(ctx, req)
	return models.CreateWebhooksResponse{}, err
}

func (p Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	if p.client == nil {
		if err := p.createClient(req.Config); err != nil {
			return models.TranslateWebhookResponse{}, err
		}
	}

	// Mangopay does not send us the event inside the body, but using
	// URL query.
	eventType, ok := req.Webhook.QueryValues["EventType"]
	if !ok || len(eventType) == 0 {
		return models.TranslateWebhookResponse{}, fmt.Errorf("missing EventType query parameter: %w", models.ErrInvalidRequest)
	}
	resourceID, ok := req.Webhook.QueryValues["RessourceId"]
	if !ok || len(resourceID) == 0 {
		return models.TranslateWebhookResponse{}, fmt.Errorf("missing RessourceId query parameter: %w", models.ErrInvalidRequest)
	}
	v, ok := req.Webhook.QueryValues["Date"]
	if !ok || len(v) == 0 {
		return models.TranslateWebhookResponse{}, fmt.Errorf("missing Date query parameter: %w", models.ErrInvalidRequest)
	}
	date, err := strconv.ParseInt(v[0], 10, 64)
	if err != nil {
		return models.TranslateWebhookResponse{}, fmt.Errorf("invalid Date query parameter: %w", models.ErrInvalidRequest)
	}

	webhook := client.Webhook{
		ResourceID: resourceID[0],
		Date:       date,
		EventType:  client.EventType(eventType[0]),
	}

	config, ok := webhookConfigs[webhook.EventType]
	if !ok {
		return models.TranslateWebhookResponse{}, fmt.Errorf("unsupported webhook event type: %w", models.ErrInvalidRequest)
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
