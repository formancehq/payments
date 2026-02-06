package connector

import (
	"context"
	"sync/atomic"
)

// BasePlugin provides a default implementation of the Plugin interface.
// All methods return ErrNotImplemented except TrimWebhook which returns
// the webhook as-is. Connector implementations can embed this struct
// and override only the methods they need.
type BasePlugin struct {
	isScheduledForDeletion atomic.Bool
}

// NewBasePlugin creates a new BasePlugin instance.
func NewBasePlugin() Plugin {
	return &BasePlugin{}
}

func (bp *BasePlugin) Name() string {
	return "default"
}

func (bp *BasePlugin) IsScheduledForDeletion() bool {
	return bp.isScheduledForDeletion.Load()
}

func (bp *BasePlugin) ScheduleForDeletion(isScheduledForDeletion bool) {
	bp.isScheduledForDeletion.Store(isScheduledForDeletion)
}

func (bp *BasePlugin) Config() PluginInternalConfig {
	return struct{}{}
}

func (bp *BasePlugin) Install(ctx context.Context, req InstallRequest) (InstallResponse, error) {
	return InstallResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) Uninstall(ctx context.Context, req UninstallRequest) (UninstallResponse, error) {
	return UninstallResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) FetchNextAccounts(ctx context.Context, req FetchNextAccountsRequest) (FetchNextAccountsResponse, error) {
	return FetchNextAccountsResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) FetchNextBalances(ctx context.Context, req FetchNextBalancesRequest) (FetchNextBalancesResponse, error) {
	return FetchNextBalancesResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) FetchNextExternalAccounts(ctx context.Context, req FetchNextExternalAccountsRequest) (FetchNextExternalAccountsResponse, error) {
	return FetchNextExternalAccountsResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) FetchNextPayments(ctx context.Context, req FetchNextPaymentsRequest) (FetchNextPaymentsResponse, error) {
	return FetchNextPaymentsResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) FetchNextOthers(ctx context.Context, req FetchNextOthersRequest) (FetchNextOthersResponse, error) {
	return FetchNextOthersResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) CreateBankAccount(ctx context.Context, req CreateBankAccountRequest) (CreateBankAccountResponse, error) {
	return CreateBankAccountResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) CreateTransfer(ctx context.Context, req CreateTransferRequest) (CreateTransferResponse, error) {
	return CreateTransferResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) ReverseTransfer(ctx context.Context, req ReverseTransferRequest) (ReverseTransferResponse, error) {
	return ReverseTransferResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) PollTransferStatus(ctx context.Context, req PollTransferStatusRequest) (PollTransferStatusResponse, error) {
	return PollTransferStatusResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) CreatePayout(ctx context.Context, req CreatePayoutRequest) (CreatePayoutResponse, error) {
	return CreatePayoutResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) ReversePayout(ctx context.Context, req ReversePayoutRequest) (ReversePayoutResponse, error) {
	return ReversePayoutResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) PollPayoutStatus(ctx context.Context, req PollPayoutStatusRequest) (PollPayoutStatusResponse, error) {
	return PollPayoutStatusResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) CreateWebhooks(ctx context.Context, req CreateWebhooksRequest) (CreateWebhooksResponse, error) {
	return CreateWebhooksResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) TrimWebhook(ctx context.Context, req TrimWebhookRequest) (TrimWebhookResponse, error) {
	// Base implementation is to return the webhook as is
	return TrimWebhookResponse{
		Webhooks: []PSPWebhook{req.Webhook},
	}, nil
}

func (bp *BasePlugin) VerifyWebhook(ctx context.Context, req VerifyWebhookRequest) (VerifyWebhookResponse, error) {
	return VerifyWebhookResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) TranslateWebhook(ctx context.Context, req TranslateWebhookRequest) (TranslateWebhookResponse, error) {
	return TranslateWebhookResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) CreateUser(ctx context.Context, req CreateUserRequest) (CreateUserResponse, error) {
	return CreateUserResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) CreateUserLink(ctx context.Context, req CreateUserLinkRequest) (CreateUserLinkResponse, error) {
	return CreateUserLinkResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) CompleteUserLink(ctx context.Context, req CompleteUserLinkRequest) (CompleteUserLinkResponse, error) {
	return CompleteUserLinkResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) UpdateUserLink(ctx context.Context, req UpdateUserLinkRequest) (UpdateUserLinkResponse, error) {
	return UpdateUserLinkResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) CompleteUpdateUserLink(ctx context.Context, req CompleteUpdateUserLinkRequest) (CompleteUpdateUserLinkResponse, error) {
	return CompleteUpdateUserLinkResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) DeleteUserConnection(ctx context.Context, req DeleteUserConnectionRequest) (DeleteUserConnectionResponse, error) {
	return DeleteUserConnectionResponse{}, ErrNotImplemented
}

func (bp *BasePlugin) DeleteUser(ctx context.Context, req DeleteUserRequest) (DeleteUserResponse, error) {
	return DeleteUserResponse{}, ErrNotImplemented
}

var _ Plugin = &BasePlugin{}
