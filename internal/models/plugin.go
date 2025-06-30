package models

import (
	"context"
	"encoding/json"
	"time"
)

const DefaultConnectorClientTimeout = 3 * time.Second

type PluginType int

const (
	PluginTypePSP PluginType = iota
	PluginTypeBankingBridge
	PluginTypeBoth
)

//go:generate mockgen -source plugin.go -destination plugin_generated.go -package models . Plugin
type Plugin interface {
	PSPPlugin
	BankingBridgePlugin

	// Common methods
	Name() string
	Install(context.Context, InstallRequest) (InstallResponse, error)
	Uninstall(context.Context, UninstallRequest) (UninstallResponse, error)

	CreateWebhooks(context.Context, CreateWebhooksRequest) (CreateWebhooksResponse, error)
	TrimWebhook(context.Context, TrimWebhookRequest) (TrimWebhookResponse, error)
	VerifyWebhook(context.Context, VerifyWebhookRequest) (VerifyWebhookResponse, error)
	TranslateWebhook(context.Context, TranslateWebhookRequest) (TranslateWebhookResponse, error)
}

type InstallRequest struct {
	ConnectorID string
}

type InstallResponse struct {
	Workflow ConnectorTasksTree
}

type UninstallRequest struct {
	ConnectorID    string
	WebhookConfigs []PSPWebhookConfig
}

type UninstallResponse struct{}

type CreateWebhooksRequest struct {
	FromPayload    json.RawMessage
	ConnectorID    string
	WebhookBaseUrl string
}

type CreateWebhooksResponse struct {
	Configs []PSPWebhookConfig
	Others  []PSPOther // used by plugin workflow
}

type TrimWebhookRequest struct {
	Webhook PSPWebhook
	Config  *WebhookConfig
}

type TrimWebhookResponse struct {
	Webhook PSPWebhook
}

type VerifyWebhookRequest struct {
	Webhook PSPWebhook
	Config  *WebhookConfig
}

type VerifyWebhookResponse struct {
	WebhookIdempotencyKey *string
}

type TranslateWebhookRequest struct {
	Name    string
	Webhook PSPWebhook
	Config  *WebhookConfig
}

type TranslateWebhookResponse struct {
	Responses []WebhookResponse
}

type WebhookResponse struct {
	Account         *PSPAccount
	ExternalAccount *PSPAccount
	Payment         *PSPPayment

	// Webhooks related to banking bridges
	TransactionReadyToFetch *TransactionReadyToFetch
}

type TransactionReadyToFetch struct {
	ID          *string
	FromPayload json.RawMessage
}

type BankBridgeFromPayload struct {
	PSUBankBridge           *PSUBankBridge
	PSUBankBridgeConnection *PSUBankBridgeConnection
	FromPayload             json.RawMessage
}
