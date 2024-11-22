package models

import (
	"context"
	"encoding/json"
)

type PluginConstructorFn func() Plugin

//go:generate mockgen -source plugin.go -destination plugin_generated.go -package models . Plugin
type Plugin interface {
	Name() string

	Install(context.Context, InstallRequest) (InstallResponse, error)
	Uninstall(context.Context, UninstallRequest) (UninstallResponse, error)

	FetchNextAccounts(context.Context, FetchNextAccountsRequest) (FetchNextAccountsResponse, error)
	FetchNextPayments(context.Context, FetchNextPaymentsRequest) (FetchNextPaymentsResponse, error)
	FetchNextBalances(context.Context, FetchNextBalancesRequest) (FetchNextBalancesResponse, error)
	FetchNextExternalAccounts(context.Context, FetchNextExternalAccountsRequest) (FetchNextExternalAccountsResponse, error)
	FetchNextOthers(context.Context, FetchNextOthersRequest) (FetchNextOthersResponse, error)

	CreateBankAccount(context.Context, CreateBankAccountRequest) (CreateBankAccountResponse, error)
	CreateTransfer(context.Context, CreateTransferRequest) (CreateTransferResponse, error)
	PollTransferStatus(context.Context, PollTransferStatusRequest) (PollTransferStatusResponse, error)
	CreatePayout(context.Context, CreatePayoutRequest) (CreatePayoutResponse, error)
	PollPayoutStatus(context.Context, PollPayoutStatusRequest) (PollPayoutStatusResponse, error)

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
	Config      json.RawMessage
	ConnectorID string
}

type UninstallResponse struct{}

type FetchNextAccountsRequest struct {
	Config      json.RawMessage
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
	Config      json.RawMessage
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
	Config      json.RawMessage
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
	Config      json.RawMessage
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
	Config      json.RawMessage
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
	Config      json.RawMessage
	BankAccount BankAccount
}

type CreateBankAccountResponse struct {
	RelatedAccount PSPAccount
}

type CreateWebhooksRequest struct {
	Config         json.RawMessage
	FromPayload    json.RawMessage
	ConnectorID    string
	WebhookBaseUrl string
}

type CreateWebhooksResponse struct {
	Others []PSPOther
}

type TranslateWebhookRequest struct {
	Config  json.RawMessage
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
	Config            json.RawMessage
	PaymentInitiation PSPPaymentInitiation
}

type CreateTransferResponse struct {
	// If payment is immediately available, it will be return here and
	// the workflow will be terminated
	Payment *PSPPayment
	// Otherwise, the payment will be nil and the transfer ID will be returned
	// to be polled regularly until the payment is available
	PollingTransferID *string
}

type PollTransferStatusRequest struct {
	Config     json.RawMessage
	TransferID string
}

type PollTransferStatusResponse struct {
	// If nil, the payment is not yet available and the function will be called
	// again later
	// If not, the payment is available and the workflow will be terminated
	Payment *PSPPayment

	// If not nil, it means that the transfer failed, the payment initiation
	// will be marked as fail and the workflow will be terminated
	Error error
}

type CreatePayoutRequest struct {
	Config            json.RawMessage
	PaymentInitiation PSPPaymentInitiation
}

type CreatePayoutResponse struct {
	// If payment is immediately available, it will be return here and
	// the workflow will be terminated
	Payment *PSPPayment
	// Otherwise, the payment will be nil and the payout ID will be returned
	// to be polled regularly until the payment is available
	PollingPayoutID *string
}

type PollPayoutStatusRequest struct {
	Config   json.RawMessage
	PayoutID string
}

type PollPayoutStatusResponse struct {
	// If nil, the payment is not yet available and the function will be called
	// again later
	// If not, the payment is available and the workflow will be terminated
	Payment *PSPPayment

	// If not nil, it means that the transfer failed, the payment initiation
	// will be marked as fail and the workflow will be terminated
	Error error
}
