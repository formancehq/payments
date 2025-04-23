package models_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskMarshalUnmarshal(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	connectorID := models.ConnectorID{
		Provider:  "stripe",
		Reference: uuid.New(),
	}
	taskID := models.TaskID{
		Reference:   "task123",
		ConnectorID: connectorID,
	}
	createdObjectID := "obj123"
	taskError := errors.New("test error")

	task := models.Task{
		ID:              taskID,
		ConnectorID:     &connectorID,
		Status:          models.TASK_STATUS_SUCCEEDED,
		CreatedAt:       now,
		UpdatedAt:       now.Add(time.Hour),
		CreatedObjectID: &createdObjectID,
		Error:           taskError,
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var unmarshaledTask models.Task
	err = json.Unmarshal(data, &unmarshaledTask)
	require.NoError(t, err)

	assert.Equal(t, task.ID.String(), unmarshaledTask.ID.String())
	assert.Equal(t, task.ConnectorID.String(), unmarshaledTask.ConnectorID.String())
	assert.Equal(t, task.Status, unmarshaledTask.Status)
	assert.Equal(t, task.CreatedAt, unmarshaledTask.CreatedAt)
	assert.Equal(t, task.UpdatedAt, unmarshaledTask.UpdatedAt)
	assert.Equal(t, *task.CreatedObjectID, *unmarshaledTask.CreatedObjectID)
	assert.Equal(t, task.Error.Error(), unmarshaledTask.Error.Error())

	task = models.Task{
		ID:        taskID,
		Status:    models.TASK_STATUS_PROCESSING,
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err = json.Marshal(task)
	require.NoError(t, err)

	err = json.Unmarshal(data, &unmarshaledTask)
	require.NoError(t, err)

	assert.Equal(t, task.ID.String(), unmarshaledTask.ID.String())
	assert.Nil(t, unmarshaledTask.ConnectorID)
	assert.Equal(t, task.Status, unmarshaledTask.Status)
	assert.Nil(t, unmarshaledTask.CreatedObjectID)
	assert.Nil(t, unmarshaledTask.Error)

	invalidJSON := []byte(`{"id": "invalid-task-id", "status": "PROCESSING", "createdAt": "2023-01-01T00:00:00Z", "updatedAt": "2023-01-01T00:00:00Z"}`)
	err = json.Unmarshal(invalidJSON, &unmarshaledTask)
	assert.Error(t, err)

	invalidJSON = []byte(`{"id": "` + taskID.String() + `", "connectorID": "invalid-connector-id", "status": "PROCESSING", "createdAt": "2023-01-01T00:00:00Z", "updatedAt": "2023-01-01T00:00:00Z"}`)
	err = json.Unmarshal(invalidJSON, &unmarshaledTask)
	assert.Error(t, err)
}

func TestTaskStatus(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "PROCESSING", string(models.TASK_STATUS_PROCESSING))
	assert.Equal(t, "SUCCEEDED", string(models.TASK_STATUS_SUCCEEDED))
	assert.Equal(t, "FAILED", string(models.TASK_STATUS_FAILED))
}
