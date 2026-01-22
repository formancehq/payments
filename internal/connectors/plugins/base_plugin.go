package plugins

import (
	"context"
	"sync/atomic"

	"github.com/formancehq/payments/internal/models"
)

type basePlugin struct {
	isScheduledForDeletion atomic.Bool
}

func NewBasePlugin() models.Plugin {
	return &basePlugin{}
}

func (dp *basePlugin) Name() string {
	return "default"
}

func (dp *basePlugin) IsScheduledForDeletion() bool {
	return dp.isScheduledForDeletion.Load()
}

func (dp *basePlugin) ScheduleForDeletion(isScheduledForDeletion bool) {
	dp.isScheduledForDeletion.Store(isScheduledForDeletion)
}

func (dp *basePlugin) Config() models.PluginInternalConfig {
	return struct{}{}
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

func (dp *basePlugin) FetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	return models.FetchNextOrdersResponse{}, ErrNotImplemented
}

func (dp *basePlugin) FetchNextConversions(ctx context.Context, req models.FetchNextConversionsRequest) (models.FetchNextConversionsResponse, error) {
	return models.FetchNextConversionsResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreateOrder(ctx context.Context, req models.CreateOrderRequest) (models.CreateOrderResponse, error) {
	return models.CreateOrderResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CancelOrder(ctx context.Context, req models.CancelOrderRequest) (models.CancelOrderResponse, error) {
	return models.CancelOrderResponse{}, ErrNotImplemented
}

func (dp *basePlugin) PollOrderStatus(ctx context.Context, req models.PollOrderStatusRequest) (models.PollOrderStatusResponse, error) {
	return models.PollOrderStatusResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreateConversion(ctx context.Context, req models.CreateConversionRequest) (models.CreateConversionResponse, error) {
	return models.CreateConversionResponse{}, ErrNotImplemented
}

func (dp *basePlugin) GetOrderBook(ctx context.Context, req models.GetOrderBookRequest) (models.GetOrderBookResponse, error) {
	return models.GetOrderBookResponse{}, ErrNotImplemented
}

func (dp *basePlugin) GetQuote(ctx context.Context, req models.GetQuoteRequest) (models.GetQuoteResponse, error) {
	return models.GetQuoteResponse{}, ErrNotImplemented
}

func (dp *basePlugin) GetTradableAssets(ctx context.Context, req models.GetTradableAssetsRequest) (models.GetTradableAssetsResponse, error) {
	return models.GetTradableAssetsResponse{}, ErrNotImplemented
}

func (dp *basePlugin) GetTicker(ctx context.Context, req models.GetTickerRequest) (models.GetTickerResponse, error) {
	return models.GetTickerResponse{}, ErrNotImplemented
}

func (dp *basePlugin) GetOHLC(ctx context.Context, req models.GetOHLCRequest) (models.GetOHLCResponse, error) {
	return models.GetOHLCResponse{}, ErrNotImplemented
}

func (dp *basePlugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	return models.CreateWebhooksResponse{}, ErrNotImplemented
}

func (dp *basePlugin) TrimWebhook(ctx context.Context, req models.TrimWebhookRequest) (models.TrimWebhookResponse, error) {
	// Base implementation is to return the webhook as is
	return models.TrimWebhookResponse{
		Webhooks: []models.PSPWebhook{req.Webhook},
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
