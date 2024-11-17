package grpc

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/grpc/proto/services"
)

type GRPCClient struct {
	client services.PluginClient
}

func (c *GRPCClient) Install(ctx context.Context, req *services.InstallRequest) (*services.InstallResponse, error) {
	return c.client.Install(ctx, req)
}

func (c *GRPCClient) Uninstall(ctx context.Context, req *services.UninstallRequest) (*services.UninstallResponse, error) {
	return c.client.Uninstall(ctx, req)
}

func (c *GRPCClient) FetchNextAccounts(ctx context.Context, req *services.FetchNextAccountsRequest) (*services.FetchNextAccountsResponse, error) {
	return c.client.FetchNextAccounts(ctx, req)
}

func (c *GRPCClient) FetchNextPayments(ctx context.Context, req *services.FetchNextPaymentsRequest) (*services.FetchNextPaymentsResponse, error) {
	return c.client.FetchNextPayments(ctx, req)
}

func (c *GRPCClient) FetchNextExternalAccounts(ctx context.Context, req *services.FetchNextExternalAccountsRequest) (*services.FetchNextExternalAccountsResponse, error) {
	return c.client.FetchNextExternalAccounts(ctx, req)
}

func (c *GRPCClient) FetchNextBalances(ctx context.Context, req *services.FetchNextBalancesRequest) (*services.FetchNextBalancesResponse, error) {
	return c.client.FetchNextBalances(ctx, req)
}

func (c *GRPCClient) FetchNextOthers(ctx context.Context, req *services.FetchNextOthersRequest) (*services.FetchNextOthersResponse, error) {
	return c.client.FetchNextOthers(ctx, req)
}

func (c *GRPCClient) CreateBankAccount(ctx context.Context, req *services.CreateBankAccountRequest) (*services.CreateBankAccountResponse, error) {
	return c.client.CreateBankAccount(ctx, req)
}

func (c *GRPCClient) CreateTransfer(ctx context.Context, req *services.CreateTransferRequest) (*services.CreateTransferResponse, error) {
	return c.client.CreateTransfer(ctx, req)
}

func (c *GRPCClient) ReverseTransfer(ctx context.Context, req *services.ReverseTransferRequest) (*services.ReverseTransferResponse, error) {
	return c.client.ReverseTransfer(ctx, req)
}

func (c *GRPCClient) PollTransferStatus(ctx context.Context, req *services.PollTransferStatusRequest) (*services.PollTransferStatusResponse, error) {
	return c.client.PollTransferStatus(ctx, req)
}

func (c *GRPCClient) CreatePayout(ctx context.Context, req *services.CreatePayoutRequest) (*services.CreatePayoutResponse, error) {
	return c.client.CreatePayout(ctx, req)
}

func (c *GRPCClient) ReversePayout(ctx context.Context, req *services.ReversePayoutRequest) (*services.ReversePayoutResponse, error) {
	return c.client.ReversePayout(ctx, req)
}

func (c *GRPCClient) PollPayoutStatus(ctx context.Context, req *services.PollPayoutStatusRequest) (*services.PollPayoutStatusResponse, error) {
	return c.client.PollPayoutStatus(ctx, req)
}

func (c *GRPCClient) CreateWebhooks(ctx context.Context, req *services.CreateWebhooksRequest) (*services.CreateWebhooksResponse, error) {
	return c.client.CreateWebhooks(ctx, req)
}

func (c *GRPCClient) TranslateWebhook(ctx context.Context, req *services.TranslateWebhookRequest) (*services.TranslateWebhookResponse, error) {
	return c.client.TranslateWebhook(ctx, req)
}

var _ PSP = &GRPCClient{}
