package plugins

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

type defaultPlugin struct{}

func NewDefaultPlugin() models.Plugin {
	return &defaultPlugin{}
}

func (dp *defaultPlugin) Name() string {
	return "default"
}

func (dp *defaultPlugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	return models.FetchNextAccountsResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	return models.FetchNextBalancesResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	return models.FetchNextExternalAccountsResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	return models.FetchNextPaymentsResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	return models.FetchNextOthersResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	return models.CreateBankAccountResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	return models.CreateTransferResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) ReverseTransfer(ctx context.Context, req models.ReverseTransferRequest) (models.ReverseTransferResponse, error) {
	return models.ReverseTransferResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	return models.PollTransferStatusResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	return models.CreatePayoutResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) ReversePayout(ctx context.Context, req models.ReversePayoutRequest) (models.ReversePayoutResponse, error) {
	return models.ReversePayoutResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	return models.PollPayoutStatusResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	return models.CreateWebhooksResponse{}, ErrNotImplemented
}

func (dp *defaultPlugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	return models.TranslateWebhookResponse{}, ErrNotImplemented
}

var _ models.Plugin = &defaultPlugin{}
