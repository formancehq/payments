package connector

import (
	"time"

	"github.com/formancehq/payments/internal/models"
)

// DefaultConnectorClientTimeout is the default timeout for connector HTTP clients.
const DefaultConnectorClientTimeout = models.DefaultConnectorClientTimeout

// Activity timeout constants.
const (
	ActivityStartToCloseTimeoutMinutesDefault = models.ActivityStartToCloseTimeoutMinutesDefault
	ActivityStartToCloseTimeoutMinutesLong    = models.ActivityStartToCloseTimeoutMinutesLong
)

// PluginType represents the type of plugin.
type PluginType = models.PluginType

const (
	PluginTypePSP         = models.PluginTypePSP
	PluginTypeOpenBanking = models.PluginTypeOpenBanking
	PluginTypeBoth        = models.PluginTypeBoth
)

// PluginInternalConfig is a generic interface for connector-specific configuration.
type PluginInternalConfig = models.PluginInternalConfig

// Plugin is the main interface that all connectors must implement.
type Plugin = models.Plugin

// PSPPlugin interface for PSP-specific operations.
type PSPPlugin = models.PSPPlugin

// OpenBankingPlugin interface for open banking operations.
type OpenBankingPlugin = models.OpenBankingPlugin

// ConnectorID identifies a connector instance.
type ConnectorID = models.ConnectorID

// Install/Uninstall types.
type (
	InstallRequest   = models.InstallRequest
	InstallResponse  = models.InstallResponse
	UninstallRequest = models.UninstallRequest
	UninstallResponse = models.UninstallResponse
)

// Webhook types.
type (
	CreateWebhooksRequest    = models.CreateWebhooksRequest
	CreateWebhooksResponse   = models.CreateWebhooksResponse
	TrimWebhookRequest       = models.TrimWebhookRequest
	TrimWebhookResponse      = models.TrimWebhookResponse
	VerifyWebhookRequest     = models.VerifyWebhookRequest
	VerifyWebhookResponse    = models.VerifyWebhookResponse
	TranslateWebhookRequest  = models.TranslateWebhookRequest
	TranslateWebhookResponse = models.TranslateWebhookResponse
	WebhookResponse          = models.WebhookResponse
	WebhookConfig            = models.WebhookConfig
)

// PSP fetch request/response types.
type (
	FetchNextAccountsRequest          = models.FetchNextAccountsRequest
	FetchNextAccountsResponse         = models.FetchNextAccountsResponse
	FetchNextExternalAccountsRequest  = models.FetchNextExternalAccountsRequest
	FetchNextExternalAccountsResponse = models.FetchNextExternalAccountsResponse
	FetchNextPaymentsRequest          = models.FetchNextPaymentsRequest
	FetchNextPaymentsResponse         = models.FetchNextPaymentsResponse
	FetchNextOthersRequest            = models.FetchNextOthersRequest
	FetchNextOthersResponse           = models.FetchNextOthersResponse
	FetchNextBalancesRequest          = models.FetchNextBalancesRequest
	FetchNextBalancesResponse         = models.FetchNextBalancesResponse
)

// Bank account types.
type (
	CreateBankAccountRequest  = models.CreateBankAccountRequest
	CreateBankAccountResponse = models.CreateBankAccountResponse
	BankAccount               = models.BankAccount
)

// Transfer types.
type (
	CreateTransferRequest      = models.CreateTransferRequest
	CreateTransferResponse     = models.CreateTransferResponse
	ReverseTransferRequest     = models.ReverseTransferRequest
	ReverseTransferResponse    = models.ReverseTransferResponse
	PollTransferStatusRequest  = models.PollTransferStatusRequest
	PollTransferStatusResponse = models.PollTransferStatusResponse
)

// Payout types.
type (
	CreatePayoutRequest      = models.CreatePayoutRequest
	CreatePayoutResponse     = models.CreatePayoutResponse
	ReversePayoutRequest     = models.ReversePayoutRequest
	ReversePayoutResponse    = models.ReversePayoutResponse
	PollPayoutStatusRequest  = models.PollPayoutStatusRequest
	PollPayoutStatusResponse = models.PollPayoutStatusResponse
)

// Open banking user types.
type (
	CreateUserRequest              = models.CreateUserRequest
	CreateUserResponse             = models.CreateUserResponse
	CreateUserLinkRequest          = models.CreateUserLinkRequest
	CreateUserLinkResponse         = models.CreateUserLinkResponse
	UpdateUserLinkRequest          = models.UpdateUserLinkRequest
	UpdateUserLinkResponse         = models.UpdateUserLinkResponse
	CompleteUserLinkRequest        = models.CompleteUserLinkRequest
	CompleteUserLinkResponse       = models.CompleteUserLinkResponse
	CompleteUpdateUserLinkRequest  = models.CompleteUpdateUserLinkRequest
	CompleteUpdateUserLinkResponse = models.CompleteUpdateUserLinkResponse
	UserLinkSuccessResponse        = models.UserLinkSuccessResponse
	UserLinkErrorResponse          = models.UserLinkErrorResponse
	DeleteUserConnectionRequest    = models.DeleteUserConnectionRequest
	DeleteUserConnectionResponse   = models.DeleteUserConnectionResponse
	DeleteUserRequest              = models.DeleteUserRequest
	DeleteUserResponse             = models.DeleteUserResponse
	HTTPCallInformation            = models.HTTPCallInformation
	CallbackState                       = models.CallbackState
	OpenBankingForwardedUserFromPayload = models.OpenBankingForwardedUserFromPayload
	OpenBankingConnectionAttempt        = models.OpenBankingConnectionAttempt
	OpenBankingForwardedUser            = models.OpenBankingForwardedUser
	OpenBankingConnection               = models.OpenBankingConnection
)

// Open banking constants.
const NoRedirectQueryParamID = models.NoRedirectQueryParamID

// Open banking callback helpers.
var CallbackStateFromString = models.CallbackStateFromString

// Ensure DefaultConnectorClientTimeout has the expected value at compile time.
var _ time.Duration = DefaultConnectorClientTimeout
