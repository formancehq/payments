package client

import (
	"context"
	"net/http"
	"net/url"
)

func (c *client) GetCapabilities(ctx context.Context) (*CapabilitiesResponse, error) {
	out := &CapabilitiesResponse{}
	return out, c.do(ctx, http.MethodGet, c.url("/v1/capabilities"), "", nil, out)
}

func (c *client) ListAccounts(ctx context.Context, p Pagination) (*AccountsPage, error) {
	out := &AccountsPage{}
	return out, c.do(ctx, http.MethodGet, c.addPagination(c.url("/v1/accounts"), p), "", nil, out)
}

func (c *client) ListExternalAccounts(ctx context.Context, p Pagination) (*AccountsPage, error) {
	out := &AccountsPage{}
	return out, c.do(ctx, http.MethodGet, c.addPagination(c.url("/v1/external-accounts"), p), "", nil, out)
}

func (c *client) GetBalances(ctx context.Context, accountID string) (*BalancesResponse, error) {
	out := &BalancesResponse{}
	return out, c.do(ctx, http.MethodGet, c.url("/v1/accounts/"+url.PathEscape(accountID)+"/balances"), "", nil, out)
}

func (c *client) ListPayments(ctx context.Context, p Pagination) (*PaymentsPage, error) {
	out := &PaymentsPage{}
	return out, c.do(ctx, http.MethodGet, c.addPagination(c.url("/v1/payments"), p), "", nil, out)
}

func (c *client) ListOrders(ctx context.Context, p Pagination) (*OrdersPage, error) {
	out := &OrdersPage{}
	return out, c.do(ctx, http.MethodGet, c.addPagination(c.url("/v1/orders"), p), "", nil, out)
}

func (c *client) ListConversions(ctx context.Context, p Pagination) (*ConversionsPage, error) {
	out := &ConversionsPage{}
	return out, c.do(ctx, http.MethodGet, c.addPagination(c.url("/v1/conversions"), p), "", nil, out)
}

func (c *client) ListOthers(ctx context.Context, name string, p Pagination) (*OthersPage, error) {
	out := &OthersPage{}
	return out, c.do(ctx, http.MethodGet, c.addPagination(c.url("/v1/others/"+url.PathEscape(name)), p), "", nil, out)
}

func (c *client) CreatePayout(ctx context.Context, idemKey string, req *PayoutRequest) (*PayoutResponse, error) {
	out := &PayoutResponse{}
	return out, c.do(ctx, http.MethodPost, c.url("/v1/payouts"), idemKey, req, out)
}

func (c *client) GetPayout(ctx context.Context, id string) (*PayoutResponse, error) {
	out := &PayoutResponse{}
	return out, c.do(ctx, http.MethodGet, c.url("/v1/payouts/"+url.PathEscape(id)), "", nil, out)
}

func (c *client) ReversePayout(ctx context.Context, idemKey, id string, req *ReverseRequest) (*PayoutResponse, error) {
	out := &PayoutResponse{}
	return out, c.do(ctx, http.MethodPost, c.url("/v1/payouts/"+url.PathEscape(id)+"/reverse"), idemKey, req, out)
}

func (c *client) CreateTransfer(ctx context.Context, idemKey string, req *TransferRequest) (*TransferResponse, error) {
	out := &TransferResponse{}
	return out, c.do(ctx, http.MethodPost, c.url("/v1/transfers"), idemKey, req, out)
}

func (c *client) GetTransfer(ctx context.Context, id string) (*TransferResponse, error) {
	out := &TransferResponse{}
	return out, c.do(ctx, http.MethodGet, c.url("/v1/transfers/"+url.PathEscape(id)), "", nil, out)
}

func (c *client) ReverseTransfer(ctx context.Context, idemKey, id string, req *ReverseRequest) (*TransferResponse, error) {
	out := &TransferResponse{}
	return out, c.do(ctx, http.MethodPost, c.url("/v1/transfers/"+url.PathEscape(id)+"/reverse"), idemKey, req, out)
}

func (c *client) CreateBankAccount(ctx context.Context, idemKey string, req *BankAccountRequest) (*BankAccountResponse, error) {
	out := &BankAccountResponse{}
	return out, c.do(ctx, http.MethodPost, c.url("/v1/bank-accounts"), idemKey, req, out)
}

func (c *client) CreateWebhookSubscription(ctx context.Context, idemKey string, req *WebhookSubscriptionRequest) (*WebhookSubscriptionResponse, error) {
	out := &WebhookSubscriptionResponse{}
	return out, c.do(ctx, http.MethodPost, c.url("/v1/webhooks"), idemKey, req, out)
}

func (c *client) DeleteWebhookSubscription(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, c.url("/v1/webhooks/"+url.PathEscape(id)), "", nil, nil)
}
