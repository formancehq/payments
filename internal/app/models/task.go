package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/uptrace/bun"

	"github.com/google/uuid"
)

type Task struct {
	bun.BaseModel `bun:"tasks.task"`

	ID          uuid.UUID `bun:",pk,nullzero"`
	ConnectorID uuid.UUID
	CreatedAt   time.Time `bun:",nullzero"`
	UpdatedAt   time.Time `bun:",nullzero"`
	Name        string
	Descriptor  json.RawMessage
	Status      TaskStatus
	Error       string
	State       json.RawMessage

	Connector *Connector `bun:"rel:belongs-to,join:connector_id=id"`
}

type TaskStatus string

const (
	TaskStatusStopped    TaskStatus = "STOPPED"
	TaskStatusPending    TaskStatus = "PENDING"
	TaskStatusActive     TaskStatus = "ACTIVE"
	TaskStatusTerminated TaskStatus = "TERMINATED"
	TaskStatusFailed     TaskStatus = "FAILED"
)

func (t Task) ParseDescriptor(to interface{}) error {
	err := json.Unmarshal(t.Descriptor, to)
	if err != nil {
		return fmt.Errorf("failed to parse descriptor: %w", err)
	}

	return nil
}
