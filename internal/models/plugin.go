package models

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"go.temporal.io/sdk/temporal"
)

type Plugin interface {
	Install(context.Context, InstallRequest) (InstallResponse, error)
	Uninstall(context.Context, UninstallRequest) (UninstallResponse, error)

	FetchNextAccounts(context.Context, FetchNextAccountsRequest) (FetchNextAccountsResponse, error)
	FetchNextPayments(context.Context, FetchNextPaymentsRequest) (FetchNextPaymentsResponse, error)
	FetchNextBalances(context.Context, FetchNextBalancesRequest) (FetchNextBalancesResponse, error)
	FetchNextExternalAccounts(context.Context, FetchNextExternalAccountsRequest) (FetchNextExternalAccountsResponse, error)
	FetchNextOthers(context.Context, FetchNextOthersRequest) (FetchNextOthersResponse, error)

	CreateBankAccount(context.Context, CreateBankAccountRequest) (CreateBankAccountResponse, error)

	CreateWebhooks(context.Context, CreateWebhooksRequest) (CreateWebhooksResponse, error)
	TranslateWebhook(context.Context, TranslateWebhookRequest) (TranslateWebhookResponse, error)
}

type PluginError struct {
	IsRetryable bool
	err         error
}

func NewPluginError(err error) *PluginError {
	isRetryable := true

	if errors.Is(err, httpwrapper.ErrStatusCodeClientError) {
		isRetryable = false
	}

	return &PluginError{
		IsRetryable: isRetryable,
		err:         err,
	}
}

func (e *PluginError) Error() string {
	return e.err.Error()
}

func (e *PluginError) ForbidRetry() *PluginError {
	e.IsRetryable = false
	return e
}

func (e *PluginError) TemporalError() error {
	if e.IsRetryable {
		return temporal.NewApplicationErrorWithCause(e.err.Error(), "plugin", e.err)
	}
	return temporal.NewNonRetryableApplicationError(e.err.Error(), "plugin", e.err)
}

type InstallRequest struct {
	Config json.RawMessage
}

type InstallResponse struct {
	Capabilities    []Capability
	Workflow        Tasks
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
