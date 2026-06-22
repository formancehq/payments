package models

import (
	"encoding/json"
	"time"
)

type OutboxEventStatus string

const (
	OUTBOX_STATUS_PENDING   OutboxEventStatus = "pending"
	OUTBOX_STATUS_FAILED    OutboxEventStatus = "failed"
	OUTBOX_STATUS_PROCESSED OutboxEventStatus = "processed"
)

const MaxOutboxRetries = 5

type OutboxEvent struct {
	// Primary key
	ID EventID `json:"id"`

	// Mandatory fields
	EventType string            `json:"eventType"`
	EntityID  string            `json:"entityId"`
	Payload   json.RawMessage   `json:"payload"`
	CreatedAt time.Time         `json:"createdAt"`
	Status    OutboxEventStatus `json:"status"`

	// Optional fields
	ConnectorID *ConnectorID `json:"connectorId,omitempty"`
	RetryCount  int          `json:"retryCount"`
	LastRetryAt *time.Time   `json:"lastRetryAt,omitempty"`
	Error       *string      `json:"error,omitempty"`
}
