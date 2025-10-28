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
