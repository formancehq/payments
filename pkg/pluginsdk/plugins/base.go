package plugins

import (
    "context"

    internalmodels "github.com/formancehq/payments/internal/models"
)

type basePlugin struct{}

func NewBasePlugin() internalmodels.Plugin {
    return &basePlugin{}
}

func (dp *basePlugin) Name() string {
    return "default"
}

func (dp *basePlugin) Config() internalmodels.PluginInternalConfig {
    return struct{}{}
}

func (dp *basePlugin) Install(ctx context.Context, req internalmodels.InstallRequest) (internalmodels.InstallResponse, error) {
    return internalmodels.InstallResponse{}, ErrNotImplemented
}

func (dp *basePlugin) Uninstall(ctx context.Context, req internalmodels.UninstallRequest) (internalmodels.UninstallResponse, error) {
    return internalmodels.UninstallResponse{}, ErrNotImplemented
}

func (dp *basePlugin) FetchNextAccounts(ctx context.Context, req internalmodels.FetchNextAccountsRequest) (internalmodels.FetchNextAccountsResponse, error) {
    return internalmodels.FetchNextAccountsResponse{}, ErrNotImplemented
}

func (dp *basePlugin) FetchNextBalances(ctx context.Context, req internalmodels.FetchNextBalancesRequest) (internalmodels.FetchNextBalancesResponse, error) {
    return internalmodels.FetchNextBalancesResponse{}, ErrNotImplemented
}

func (dp *basePlugin) FetchNextExternalAccounts(ctx context.Context, req internalmodels.FetchNextExternalAccountsRequest) (internalmodels.FetchNextExternalAccountsResponse, error) {
    return internalmodels.FetchNextExternalAccountsResponse{}, ErrNotImplemented
}

func (dp *basePlugin) FetchNextPayments(ctx context.Context, req internalmodels.FetchNextPaymentsRequest) (internalmodels.FetchNextPaymentsResponse, error) {
    return internalmodels.FetchNextPaymentsResponse{}, ErrNotImplemented
}

func (dp *basePlugin) FetchNextOthers(ctx context.Context, req internalmodels.FetchNextOthersRequest) (internalmodels.FetchNextOthersResponse, error) {
    return internalmodels.FetchNextOthersResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreateBankAccount(ctx context.Context, req internalmodels.CreateBankAccountRequest) (internalmodels.CreateBankAccountResponse, error) {
    return internalmodels.CreateBankAccountResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreateTransfer(ctx context.Context, req internalmodels.CreateTransferRequest) (internalmodels.CreateTransferResponse, error) {
    return internalmodels.CreateTransferResponse{}, ErrNotImplemented
}

func (dp *basePlugin) ReverseTransfer(ctx context.Context, req internalmodels.ReverseTransferRequest) (internalmodels.ReverseTransferResponse, error) {
    return internalmodels.ReverseTransferResponse{}, ErrNotImplemented
}

func (dp *basePlugin) PollTransferStatus(ctx context.Context, req internalmodels.PollTransferStatusRequest) (internalmodels.PollTransferStatusResponse, error) {
    return internalmodels.PollTransferStatusResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreatePayout(ctx context.Context, req internalmodels.CreatePayoutRequest) (internalmodels.CreatePayoutResponse, error) {
    return internalmodels.CreatePayoutResponse{}, ErrNotImplemented
}

func (dp *basePlugin) ReversePayout(ctx context.Context, req internalmodels.ReversePayoutRequest) (internalmodels.ReversePayoutResponse, error) {
    return internalmodels.ReversePayoutResponse{}, ErrNotImplemented
}

func (dp *basePlugin) PollPayoutStatus(ctx context.Context, req internalmodels.PollPayoutStatusRequest) (internalmodels.PollPayoutStatusResponse, error) {
    return internalmodels.PollPayoutStatusResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreateWebhooks(ctx context.Context, req internalmodels.CreateWebhooksRequest) (internalmodels.CreateWebhooksResponse, error) {
    return internalmodels.CreateWebhooksResponse{}, ErrNotImplemented
}

func (dp *basePlugin) TrimWebhook(ctx context.Context, req internalmodels.TrimWebhookRequest) (internalmodels.TrimWebhookResponse, error) {
    return internalmodels.TrimWebhookResponse{
        Webhooks: []internalmodels.PSPWebhook{req.Webhook},
    }, nil
}

func (dp *basePlugin) VerifyWebhook(ctx context.Context, req internalmodels.VerifyWebhookRequest) (internalmodels.VerifyWebhookResponse, error) {
    return internalmodels.VerifyWebhookResponse{}, ErrNotImplemented
}

func (dp *basePlugin) TranslateWebhook(ctx context.Context, req internalmodels.TranslateWebhookRequest) (internalmodels.TranslateWebhookResponse, error) {
    return internalmodels.TranslateWebhookResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreateUser(ctx context.Context, req internalmodels.CreateUserRequest) (internalmodels.CreateUserResponse, error) {
    return internalmodels.CreateUserResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreateUserLink(ctx context.Context, req internalmodels.CreateUserLinkRequest) (internalmodels.CreateUserLinkResponse, error) {
    return internalmodels.CreateUserLinkResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CompleteUserLink(ctx context.Context, req internalmodels.CompleteUserLinkRequest) (internalmodels.CompleteUserLinkResponse, error) {
    return internalmodels.CompleteUserLinkResponse{}, ErrNotImplemented
}

func (dp *basePlugin) UpdateUserLink(ctx context.Context, req internalmodels.UpdateUserLinkRequest) (internalmodels.UpdateUserLinkResponse, error) {
    return internalmodels.UpdateUserLinkResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CompleteUpdateUserLink(ctx context.Context, req internalmodels.CompleteUpdateUserLinkRequest) (internalmodels.CompleteUpdateUserLinkResponse, error) {
    return internalmodels.CompleteUpdateUserLinkResponse{}, ErrNotImplemented
}

func (dp *basePlugin) DeleteUserConnection(ctx context.Context, req internalmodels.DeleteUserConnectionRequest) (internalmodels.DeleteUserConnectionResponse, error) {
    return internalmodels.DeleteUserConnectionResponse{}, ErrNotImplemented
}

func (dp *basePlugin) DeleteUser(ctx context.Context, req internalmodels.DeleteUserRequest) (internalmodels.DeleteUserResponse, error) {
    return internalmodels.DeleteUserResponse{}, ErrNotImplemented
}

