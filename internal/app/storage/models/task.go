package models

import (
	"time"

	"github.com/uptrace/bun"

	"github.com/google/uuid"
)

type Task struct {
	bun.BaseModel `bun:"tasks.task"`

	ID          uuid.UUID
	ConnectorID uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string
	Descriptor  any
	Status      TaskStatus
	Error       string
	State       any

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
