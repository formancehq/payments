package dummypay

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/dummypay/client"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
)

func init() {
	registry.RegisterPlugin(registry.DummyPSPName, models.PluginTypePSP, func(_ models.ConnectorID, name string, logger logging.Logger, rm json.RawMessage) (models.Plugin, error) {
		return New(name, logger, rm)
	}, capabilities, Config{})
}

type Plugin struct {
	models.Plugin

	name   string
	config Config
	logger logging.Logger
	client client.Client
}

func New(name string, logger logging.Logger, rawConfig json.RawMessage) (*Plugin, error) {
	conf, err := unmarshalAndValidateConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	return &Plugin{
		Plugin: plugins.NewBasePlugin(),

		name:   name,
		logger: logger,
		client: client.New(conf.Directory),
		config: conf,
	}, nil
}

func (p *Plugin) Name() string {
	return p.name
}

func (p *Plugin) Config() models.PluginInternalConfig {
	return p.config
}

func (p *Plugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return p.install(ctx, req)
}

func (p *Plugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, nil
}

func (p *Plugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	return p.fetchNextAccounts(ctx, req)
}

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	return p.fetchNextBalances(ctx, req)
}

func (p *Plugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	return models.FetchNextPaymentsResponse{
		Payments: []models.PSPPayment{},
		NewState: json.RawMessage(`{}`),
		HasMore:  false,
	}, nil
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
		return models.ReversePayoutResponse{}, fmt.Errorf("failed to reverse payout using client: %w", err)
	}
	return models.ReversePayoutResponse{Payment: pspPayment}, nil
}

func (p *Plugin) CreateUser(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
	if p.client == nil {
		return models.CreateUserResponse{}, plugins.ErrNotYetInstalled
	}

	return p.createUser(ctx, req)
}

func (p *Plugin) CreateUserLink(ctx context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	if p.client == nil {
		return models.CreateUserLinkResponse{}, plugins.ErrNotYetInstalled
	}

	return p.createUserLink(ctx, req)
}

func (p *Plugin) CompleteUserLink(ctx context.Context, req models.CompleteUserLinkRequest) (models.CompleteUserLinkResponse, error) {
	if p.client == nil {
		return models.CompleteUserLinkResponse{}, plugins.ErrNotYetInstalled
	}

	return p.completeUserLink(ctx, req)
}

func (p *Plugin) UpdateUserLink(ctx context.Context, req models.UpdateUserLinkRequest) (models.UpdateUserLinkResponse, error) {
	if p.client == nil {
		return models.UpdateUserLinkResponse{}, plugins.ErrNotYetInstalled
	}

	return p.updateUserLink(ctx, req)
}

func (p *Plugin) DeleteUser(ctx context.Context, req models.DeleteUserRequest) (models.DeleteUserResponse, error) {
	if p.client == nil {
		return models.DeleteUserResponse{}, plugins.ErrNotYetInstalled
	}

	return p.deleteUser(ctx, req)
}

func (p *Plugin) DeleteUserConnection(ctx context.Context, req models.DeleteUserConnectionRequest) (models.DeleteUserConnectionResponse, error) {
	if p.client == nil {
		return models.DeleteUserConnectionResponse{}, plugins.ErrNotYetInstalled
	}

	return p.deleteUserConnection(ctx, req)
}

var _ models.Plugin = &Plugin{}
