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
	TranslateWebhook(context.Context, TranslateWebhookRequest) (TranslateWebhookResponse, error)
}

type InstallRequest struct {
	ConnectorID string
}

type InstallResponse struct {
	Workflow ConnectorTasksTree
}

type UninstallRequest struct {
	ConnectorID string
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

type TranslateWebhookRequest struct {
	Name    string
	Webhook PSPWebhook
	Config  *WebhookConfig
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
