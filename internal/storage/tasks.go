package storage

import (
	"context"
	"encoding/json"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

type task struct {
	bun.BaseModel `bun:"table:tasks"`

	// Mandatory fields
	ID        models.TaskID     `bun:"id,pk,type:character varying,notnull"`
	Status    models.TaskStatus `bun:"status,type:text,notnull"`
	CreatedAt time.Time         `bun:"created_at,type:timestamp without time zone,notnull"`
	UpdatedAt time.Time         `bun:"updated_at,type:timestamp without time zone,notnull"`

	// Optional fields
	ConnectorID     *models.ConnectorID `bun:"connector_id,type:character varying"`
	CreatedObjectID *string             `bun:"created_object_id,type:character varying"`
	Error           *string             `bun:"error,type:text"`
}

func (s *store) TasksUpsert(ctx context.Context, task models.Task) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e("failed to begin transaction", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	t := fromTaskModel(task)

	query := tx.NewInsert().
		Model(&t).
		On("CONFLICT (id) DO UPDATE").
		Set("status = EXCLUDED.status").
		Set("updated_at = EXCLUDED.updated_at")

	if task.CreatedObjectID != nil {
		query.Set("created_object_id = EXCLUDED.created_object_id")
	}

	if task.Error != nil {
		query.Set("error = EXCLUDED.error")
	} else {
		query.Set("error = NULL")
	}

	_, err = query.Exec(ctx)
	if err != nil {
		return e("failed to insert task", err)
	}

	// Create outbox event for task update
	payload := map[string]interface{}{
		"id":        task.ID.String(),
		"status":    string(task.Status),
		"createdAt": task.CreatedAt,
		"updatedAt": task.UpdatedAt,
	}
	if task.ConnectorID != nil {
		payload["connectorID"] = task.ConnectorID.String()
	}
	if task.CreatedObjectID != nil {
		payload["createdObjectID"] = *task.CreatedObjectID
	}
	if task.Error != nil {
		payload["error"] = task.Error.Error()
	}

	var payloadBytes []byte
	payloadBytes, err = json.Marshal(payload)
	if err != nil {
		return e("failed to marshal task event payload", err)
	}

	outboxEvent := models.OutboxEvent{
		EventType:      models.OUTBOX_EVENT_TASK_UPDATED,
		EntityID:       task.ID.String(),
		Payload:        payloadBytes,
		CreatedAt:      time.Now().UTC().Time,
		Status:         models.OUTBOX_STATUS_PENDING,
		ConnectorID:    task.ConnectorID,
		IdempotencyKey: task.IdempotencyKey(),
	}

	if err = s.OutboxEventsInsert(ctx, tx, []models.OutboxEvent{outboxEvent}); err != nil {
		return err
	}

	return e("failed to commit transaction", tx.Commit())
}

func (s *store) TasksGet(ctx context.Context, id models.TaskID) (*models.Task, error) {
	var t task

	err := s.db.NewSelect().
		Model(&t).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("failed to fetch task", err)
	}

	return pointer.For(toTaskModel(t)), nil
}

func (s *store) TasksDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*task)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)
	return e("failed to delete tasks", err)
}

func fromTaskModel(from models.Task) task {
	return task{
		ID:              from.ID,
		ConnectorID:     from.ConnectorID,
		Status:          from.Status,
		CreatedAt:       time.New(from.CreatedAt),
		UpdatedAt:       time.New(from.UpdatedAt),
		CreatedObjectID: from.CreatedObjectID,
		Error: func() *string {
			if from.Error == nil {
				return nil
			}
			return pointer.For(from.Error.Error())
		}(),
	}
}

func toTaskModel(from task) models.Task {
	return models.Task{
		ID:              from.ID,
		ConnectorID:     from.ConnectorID,
		Status:          from.Status,
		CreatedAt:       from.CreatedAt.Time,
		UpdatedAt:       from.UpdatedAt.Time,
		CreatedObjectID: from.CreatedObjectID,
		Error: func() error {
			if from.Error == nil {
				return nil
			}
			return errors.New(*from.Error)
		}(),
	}
}
