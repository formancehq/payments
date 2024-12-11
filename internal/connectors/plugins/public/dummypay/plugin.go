package dummypay

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/dummypay/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

func init() {
	registry.RegisterPlugin("dummypay", func(name string, rm json.RawMessage) (models.Plugin, error) {
		return New(name, rm)
	}, capabilities)
}

type Plugin struct {
	name   string
	client client.Client
}

func New(name string, rawConfig json.RawMessage) (*Plugin, error) {
	conf, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	return &Plugin{
		name:   name,
		client: client.New(conf.Directory),
	}, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return p.install(ctx, req)
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

type accountsState struct {
	NextToken int `json:"nextToken"`
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	accounts, next, err := p.client.FetchAccounts(ctx, oldState.NextToken, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to fetch accounts from client: %w", err)
	}

	newState := accountsState{
		NextToken: next,
	}
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  next > 0,
	}, nil
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	balance, err := p.client.FetchBalance(ctx, from.Reference)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to fetch balance from client: %w", err)
	}

	balances := make([]models.PSPBalance, 0, 1)
	if balance != nil {
		balances = append(balances, *balance)
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}

func (p *Plugin) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	return models.FetchNextExternalAccountsResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	return models.FetchNextPaymentsResponse{
		Payments: []models.PSPPayment{},
		NewState: json.RawMessage(`{}`),
		HasMore:  false,
	}, nil
}

func (p *Plugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	return models.FetchNextOthersResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	name := "dummypay-account"
	bankAccount := models.PSPAccount{
		Reference: fmt.Sprintf("dummypay-%s", req.BankAccount.ID.String()),
		CreatedAt: req.BankAccount.CreatedAt,
		Name:      &name,
		Raw:       json.RawMessage(`{}`),
	}
	return models.CreateBankAccountResponse{
		RelatedAccount: bankAccount,
	}, nil
}

func (p *Plugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	pspPayment, err := p.client.CreatePayment(ctx, models.PAYMENT_TYPE_TRANSFER, req.PaymentInitiation)
	if err != nil {
		return models.CreateTransferResponse{}, fmt.Errorf("failed to create transfer using client: %w", err)
	}
	return models.CreateTransferResponse{Payment: pspPayment}, nil
}

func (p *Plugin) ReverseTransfer(ctx context.Context, req models.ReverseTransferRequest) (models.ReverseTransferResponse, error) {
	pspPayment, err := p.client.ReversePayment(ctx, models.PAYMENT_TYPE_TRANSFER, req.PaymentInitiationReversal)
	if err != nil {
		return models.ReverseTransferResponse{}, fmt.Errorf("failed to reverse transfer using client: %w", err)
	}
	return models.ReverseTransferResponse{Payment: pspPayment}, nil
}

func (p *Plugin) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	return models.PollTransferStatusResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	pspPayment, err := p.client.CreatePayment(ctx, models.PAYMENT_TYPE_PAYOUT, req.PaymentInitiation)
	if err != nil {
		return models.CreatePayoutResponse{}, fmt.Errorf("failed to create transfer using client: %w", err)
	}
	return models.CreatePayoutResponse{Payment: pspPayment}, nil
}

func (p *Plugin) ReversePayout(ctx context.Context, req models.ReversePayoutRequest) (models.ReversePayoutResponse, error) {
	pspPayment, err := p.client.ReversePayment(ctx, models.PAYMENT_TYPE_PAYOUT, req.PaymentInitiationReversal)
	if err != nil {
		return models.ReversePayoutResponse{}, fmt.Errorf("failed to reverse transfer using client: %w", err)
	}
	return models.ReversePayoutResponse{Payment: pspPayment}, nil
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	return models.CreateWebhooksResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	return models.PollPayoutStatusResponse{}, plugins.ErrNotImplemented
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	return models.TranslateWebhookResponse{}, plugins.ErrNotImplemented
}

var _ models.Plugin = &Plugin{}
