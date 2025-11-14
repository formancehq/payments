package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OutboxEventStatus string

const (
	OUTBOX_STATUS_PENDING OutboxEventStatus = "pending"
	OUTBOX_STATUS_FAILED  OutboxEventStatus = "failed"
)

// Outbox event type constants
const (
	OUTBOX_EVENT_ACCOUNT_SAVED                            = "account.saved"
	OUTBOX_EVENT_BALANCE_SAVED                            = "balance.saved"
	OUTBOX_EVENT_PAYMENT_SAVED                            = "payment.saved"
	OUTBOX_EVENT_PAYMENT_DELETED                          = "payment.deleted"
	OUTBOX_EVENT_BANK_ACCOUNT_SAVED                       = "bank_account.saved"
	OUTBOX_EVENT_TASK_UPDATED                             = "task.updated"
	OUTBOX_EVENT_CONNECTOR_RESET                          = "connector.reset"
	OUTBOX_EVENT_POOL_SAVED                               = "pool.saved"
	OUTBOX_EVENT_POOL_DELETED                             = "pool.deleted"
	OUTBOX_EVENT_PAYMENT_INITIATION_SAVED                 = "payment_initiation.saved"
	OUTBOX_EVENT_PAYMENT_INITIATION_ADJUSTMENT_SAVED      = "payment_initiation_adjustment.saved"
	OUTBOX_EVENT_PAYMENT_INITIATION_RELATED_PAYMENT_SAVED = "payment_initiation_related_payment.saved"
	OUTBOX_EVENT_USER_LINK_STATUS                         = "user.link_status"
	OUTBOX_EVENT_USER_CONNECTION_DATA_SYNCED              = "user.connection_data_synced"
	OUTBOX_EVENT_USER_CONNECTION_PENDING_DISCONNECT       = "user.connection_pending_disconnect"
	OUTBOX_EVENT_USER_CONNECTION_DISCONNECTED             = "user.connection_disconnected"
	OUTBOX_EVENT_USER_CONNECTION_RECONNECTED              = "user.connection_reconnected"
	OUTBOX_EVENT_USER_DISCONNECTED                        = "user.disconnected"
)

const MaxOutboxRetries = 5

type OutboxEvent struct {
	// Autoincrement fields
	SortID int64 `json:"sortId"`

	// Mandatory fields
	ID             uuid.UUID         `json:"id"`
	EventType      string            `json:"eventType"`
	EntityID       string            `json:"entityId"`
	Payload        json.RawMessage   `json:"payload"`
	CreatedAt      time.Time         `json:"createdAt"`
	Status         OutboxEventStatus `json:"status"`
	IdempotencyKey string            `json:"idempotencyKey"`

	// Optional fields
	ConnectorID *ConnectorID `json:"connectorId,omitempty"`
	RetryCount  int          `json:"retryCount"`
	LastRetryAt *time.Time   `json:"lastRetryAt,omitempty"`
	Error       *string      `json:"error,omitempty"`
}
