package models

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const DefaultConnectorClientTimeout = 10 * time.Second

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
	Webhooks []PSPWebhook
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
	UserLinkSessionFinished         *PSPUserLinkSessionFinished
	DataReadyToFetch                *PSPDataReadyToFetch
	UserConnectionPendingDisconnect *PSPUserConnectionPendingDisconnect
	UserConnectionDisconnected      *PSPUserConnectionDisconnected
}

type PSPDataReadyToFetch struct {
	ID          *string
	FromPayload json.RawMessage
}

type PSPDataToDelete struct {
	ConnectionID string

	// If filled, the account and all its data will be deleted. The connection
	// ID will still be available in the database.
	// If not filled, all data associated with the connection will be deleted.
	AccountID *string

	FromPayload json.RawMessage
}

type PSPUserConnectionPendingDisconnect struct {
	ConnectionID string
	At           time.Time
	Reason       *string
}

type UserConnectionPendingDisconnect struct {
	PsuID        uuid.UUID
	ConnectorID  ConnectorID
	ConnectionID string
	At           time.Time
	Reason       *string
}

func (u UserConnectionPendingDisconnect) IdempotencyKey() string {
	return IdempotencyKey(u)
}

type PSPUserConnectionDisconnected struct {
	ConnectionID string
	At           time.Time
	Reason       *string
}

type UserConnectionDisconnected struct {
	PsuID        uuid.UUID
	ConnectorID  ConnectorID
	ConnectionID string
	At           time.Time
	Reason       *string
}

func (u UserConnectionDisconnected) IdempotencyKey() string {
	return IdempotencyKey(u)
}

type PSPUserLinkSessionFinished struct {
	AttemptID uuid.UUID
	Status    PSUBankBridgeConnectionAttemptStatus
	Error     *string
}

type UserLinkSessionFinished struct {
	PsuID       uuid.UUID
	ConnectorID ConnectorID
	AttemptID   uuid.UUID
	Status      PSUBankBridgeConnectionAttemptStatus
	Error       *string
}

func (u UserLinkSessionFinished) IdempotencyKey() string {
	return IdempotencyKey(u)
}

type UserConnectionDataSynced struct {
	PsuID        uuid.UUID
	ConnectorID  ConnectorID
	ConnectionID string
	At           time.Time
}

func (u UserConnectionDataSynced) IdempotencyKey() string {
	return IdempotencyKey(u)
}

type BankBridgeFromPayload struct {
	PSUBankBridge           *PSUBankBridge
	PSUBankBridgeConnection *PSUBankBridgeConnection
	FromPayload             json.RawMessage
}
