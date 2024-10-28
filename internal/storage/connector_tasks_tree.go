package storage

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

type connectorTasksTree struct {
	bun.BaseModel `bun:"table:connector_tasks_tree"`

	// Mandatory fields
	ConnectorID models.ConnectorID `bun:"connector_id,pk,type:character varying,notnull"`
	TasksTree   json.RawMessage    `bun:"tasks,type:json,notnull"`
}

func (s *store) ConnectorTasksTreeUpsert(ctx context.Context, connectorID models.ConnectorID, ts models.ConnectorTasksTree) error {
	payload, err := json.Marshal(&ts)
	if err != nil {
		return errors.Wrap(err, "failed to marshal tasks")
	}

	tasks := connectorTasksTree{
		ConnectorID: connectorID,
		TasksTree:   payload,
	}

	_, err = s.db.NewInsert().
		Model(&tasks).
		On("CONFLICT (connector_id) DO UPDATE").
		Set("tasks = EXCLUDED.tasks").
		Exec(ctx)
	return e("failed to insert tasks", err)
}

func (s *store) ConnectorTasksTreeGet(ctx context.Context, connectorID models.ConnectorID) (*models.ConnectorTasksTree, error) {
	var ts connectorTasksTree

	err := s.db.NewSelect().
		Model(&ts).
		Where("connector_id = ?", connectorID).
		Scan(ctx)
	if err != nil {
		return nil, e("failed to fetch tasks", err)
	}

	var tasks models.ConnectorTasksTree
	if err := json.Unmarshal(ts.TasksTree, &tasks); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal tasks")
	}

	return &tasks, nil
}

func (s *store) ConnectorTasksTreeDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*connectorTasksTree)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)

	return e("failed to delete tasks", err)
}
