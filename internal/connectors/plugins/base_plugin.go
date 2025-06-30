package plugins

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

type basePlugin struct{}

func NewBasePlugin() models.Plugin {
	return &basePlugin{}
}

func (dp *basePlugin) Name() string {
	return "default"
}

func (dp *basePlugin) Install(ctx context.Context, req models.InstallRequest) (models.InstallResponse, error) {
	return models.InstallResponse{}, ErrNotImplemented
}

func (dp *basePlugin) Uninstall(ctx context.Context, req models.UninstallRequest) (models.UninstallResponse, error) {
	return models.UninstallResponse{}, ErrNotImplemented
}

func (dp *basePlugin) FetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	return models.FetchNextAccountsResponse{}, ErrNotImplemented
}

func (dp *basePlugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	return models.FetchNextBalancesResponse{}, ErrNotImplemented
}

func (dp *basePlugin) FetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	return models.FetchNextExternalAccountsResponse{}, ErrNotImplemented
}

func (dp *basePlugin) FetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	return models.FetchNextPaymentsResponse{}, ErrNotImplemented
}

func (dp *basePlugin) FetchNextOthers(ctx context.Context, req models.FetchNextOthersRequest) (models.FetchNextOthersResponse, error) {
	return models.FetchNextOthersResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreateBankAccount(ctx context.Context, req models.CreateBankAccountRequest) (models.CreateBankAccountResponse, error) {
	return models.CreateBankAccountResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	return models.CreateTransferResponse{}, ErrNotImplemented
}

func (dp *basePlugin) ReverseTransfer(ctx context.Context, req models.ReverseTransferRequest) (models.ReverseTransferResponse, error) {
	return models.ReverseTransferResponse{}, ErrNotImplemented
}

func (dp *basePlugin) PollTransferStatus(ctx context.Context, req models.PollTransferStatusRequest) (models.PollTransferStatusResponse, error) {
	return models.PollTransferStatusResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	return models.CreatePayoutResponse{}, ErrNotImplemented
}

func (dp *basePlugin) ReversePayout(ctx context.Context, req models.ReversePayoutRequest) (models.ReversePayoutResponse, error) {
	return models.ReversePayoutResponse{}, ErrNotImplemented
}

func (dp *basePlugin) PollPayoutStatus(ctx context.Context, req models.PollPayoutStatusRequest) (models.PollPayoutStatusResponse, error) {
	return models.PollPayoutStatusResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	return models.CreateWebhooksResponse{}, ErrNotImplemented
}

func (dp *basePlugin) TrimWebhook(ctx context.Context, req models.TrimWebhookRequest) (models.TrimWebhookResponse, error) {
	// Base implementation is to return the webhook as is
	return models.TrimWebhookResponse{
		Webhook: req.Webhook,
	}, nil
}

func (dp *basePlugin) VerifyWebhook(ctx context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	return models.VerifyWebhookResponse{}, ErrNotImplemented
}

func (dp *basePlugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	return models.TranslateWebhookResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreateUser(ctx context.Context, req models.CreateUserRequest) (models.CreateUserResponse, error) {
	return models.CreateUserResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreateUserLink(ctx context.Context, req models.CreateUserLinkRequest) (models.CreateUserLinkResponse, error) {
	return models.CreateUserLinkResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CompleteUserLink(ctx context.Context, req models.CompleteUserLinkRequest) (models.CompleteUserLinkResponse, error) {
	return models.CompleteUserLinkResponse{}, ErrNotImplemented
}

func (dp *basePlugin) UpdateUserLink(ctx context.Context, req models.UpdateUserLinkRequest) (models.UpdateUserLinkResponse, error) {
	return models.UpdateUserLinkResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CompleteUpdateUserLink(ctx context.Context, req models.CompleteUpdateUserLinkRequest) (models.CompleteUpdateUserLinkResponse, error) {
	return models.CompleteUpdateUserLinkResponse{}, ErrNotImplemented
}

func (dp *basePlugin) DeleteUserConnection(ctx context.Context, req models.DeleteUserConnectionRequest) (models.DeleteUserConnectionResponse, error) {
	return models.DeleteUserConnectionResponse{}, ErrNotImplemented
}

func (dp *basePlugin) DeleteUser(ctx context.Context, req models.DeleteUserRequest) (models.DeleteUserResponse, error) {
	return models.DeleteUserResponse{}, ErrNotImplemented
}

var _ models.Plugin = &basePlugin{}
