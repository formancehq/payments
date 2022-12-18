package task

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/formancehq/payments/internal/app/models"
)

type Repository interface {
	UpdateTaskStatus(ctx context.Context, provider models.ConnectorProvider, descriptor json.RawMessage, status models.TaskStatus, err string) error
	FindAndUpsertTask(ctx context.Context, provider models.ConnectorProvider, descriptor json.RawMessage, status models.TaskStatus, err string) (*models.Task, error)
	ListTasksByStatus(ctx context.Context, provider models.ConnectorProvider, status models.TaskStatus) ([]models.Task, error)
	ListTasks(ctx context.Context, provider models.ConnectorProvider) ([]models.Task, error)
	ReadOldestPendingTask(ctx context.Context, provider models.ConnectorProvider) (*models.Task, error)
	GetTask(ctx context.Context, id uuid.UUID) (*models.Task, error)
	GetTaskByDescriptor(ctx context.Context, provider models.ConnectorProvider, descriptor json.RawMessage) (*models.Task, error)
}
