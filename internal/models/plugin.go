package models

import (
	"context"
	"encoding/json"
)

type PluginConstructorFn func() Plugin

//go:generate mockgen -source plugin.go -destination plugin_generated.go -package models . Plugin
type Plugin interface {
	Install(context.Context, InstallRequest) (InstallResponse, error)
	Uninstall(context.Context, UninstallRequest) (UninstallResponse, error)

	FetchNextAccounts(context.Context, FetchNextAccountsRequest) (FetchNextAccountsResponse, error)
	FetchNextPayments(context.Context, FetchNextPaymentsRequest) (FetchNextPaymentsResponse, error)
	FetchNextBalances(context.Context, FetchNextBalancesRequest) (FetchNextBalancesResponse, error)
	FetchNextExternalAccounts(context.Context, FetchNextExternalAccountsRequest) (FetchNextExternalAccountsResponse, error)
	FetchNextOthers(context.Context, FetchNextOthersRequest) (FetchNextOthersResponse, error)

	CreateBankAccount(context.Context, CreateBankAccountRequest) (CreateBankAccountResponse, error)
	CreateTransfer(context.Context, CreateTransferRequest) (CreateTransferResponse, error)
	CreatePayout(context.Context, CreatePayoutRequest) (CreatePayoutResponse, error)

	CreateWebhooks(context.Context, CreateWebhooksRequest) (CreateWebhooksResponse, error)
	TranslateWebhook(context.Context, TranslateWebhookRequest) (TranslateWebhookResponse, error)
}

type InstallRequest struct {
	Config json.RawMessage
}

type InstallResponse struct {
	Capabilities    []Capability
	Workflow        ConnectorTasksTree
	WebhooksConfigs []PSPWebhookConfig
}

type UninstallRequest struct {
	ConnectorID string
}

type UninstallResponse struct{}

type FetchNextAccountsRequest struct {
	FromPayload json.RawMessage
	State       json.RawMessage
	PageSize    int
}

type FetchNextAccountsResponse struct {
	Accounts []PSPAccount
	NewState json.RawMessage
	HasMore  bool
}

type FetchNextExternalAccountsRequest struct {
	FromPayload json.RawMessage
	State       json.RawMessage
	PageSize    int
}

type FetchNextExternalAccountsResponse struct {
	ExternalAccounts []PSPAccount
	NewState         json.RawMessage
	HasMore          bool
}

type FetchNextPaymentsRequest struct {
	FromPayload json.RawMessage
	State       json.RawMessage
	PageSize    int
}

type FetchNextPaymentsResponse struct {
	Payments []PSPPayment
	NewState json.RawMessage
	HasMore  bool
}

type FetchNextOthersRequest struct {
	Name        string
	FromPayload json.RawMessage
	State       json.RawMessage
	PageSize    int
}

type FetchNextOthersResponse struct {
	Others   []PSPOther
	NewState json.RawMessage
	HasMore  bool
}

type FetchNextBalancesRequest struct {
	FromPayload json.RawMessage
	State       json.RawMessage
	PageSize    int
}

type FetchNextBalancesResponse struct {
	Balances []PSPBalance
	NewState json.RawMessage
	HasMore  bool
}

type CreateBankAccountRequest struct {
	BankAccount BankAccount
}

type CreateBankAccountResponse struct {
	RelatedAccount PSPAccount
}

type CreateWebhooksRequest struct {
	FromPayload    json.RawMessage
	ConnectorID    string
	WebhookBaseUrl string
}

type CreateWebhooksResponse struct {
	Others []PSPOther
}

type TranslateWebhookRequest struct {
	Name    string
	Webhook PSPWebhook
}

type WebhookResponse struct {
	IdempotencyKey  string
	Account         *PSPAccount
	ExternalAccount *PSPAccount
	Payment         *PSPPayment
}

type TranslateWebhookResponse struct {
	Responses []WebhookResponse
}

type CreateTransferRequest struct {
	PaymentInitiation PSPPaymentInitiation
}

type CreateTransferResponse struct {
	Payment PSPPayment
}

type CreatePayoutRequest struct {
	PaymentInitiation PSPPaymentInitiation
}

type CreatePayoutResponse struct {
	Payment PSPPayment
}
