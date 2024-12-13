package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
)

type TaskStatus string

const (
	TASK_STATUS_PROCESSING TaskStatus = "PROCESSING"
	TASK_STATUS_SUCCEEDED  TaskStatus = "SUCCEEDED"
	TASK_STATUS_FAILED     TaskStatus = "FAILED"
)

type Task struct {
	// Unique identifier of the task
	ID TaskID `json:"id"`
	// Related Connector ID
	ConnectorID *ConnectorID `json:"connectorID"`
	// Status of the task
	Status TaskStatus `json:"status"`
	// Time when the task was created
	CreatedAt time.Time `json:"createdAt"`
	// Time when the task was last updated
	UpdatedAt time.Time `json:"updatedAt"`

	CreatedObjectID *string `json:"createdObjectID,omitempty"`
	Error           error   `json:"error,omitempty"`
}

func (t Task) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID              string     `json:"id"`
		ConnectorID     string     `json:"connectorID"`
		Status          TaskStatus `json:"status"`
		CreatedAt       time.Time  `json:"createdAt"`
		UpdatedAt       time.Time  `json:"updatedAt"`
		CreatedObjectID *string    `json:"createdObjectID,omitempty"`
		Error           *string    `json:"error,omitempty"`
	}{
		ID:              t.ID.String(),
		ConnectorID:     t.ConnectorID.String(),
		Status:          t.Status,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
		CreatedObjectID: t.CreatedObjectID,
		Error: func() *string {
			if t.Error == nil {
				return nil
			}

			return pointer.For(t.Error.Error())
		}(),
	})
}

func (t *Task) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID              string     `json:"id"`
		ConnectorID     *string    `json:"connectorID"`
		Status          TaskStatus `json:"status"`
		CreatedAt       time.Time  `json:"createdAt"`
		UpdatedAt       time.Time  `json:"updatedAt"`
		CreatedObjectID *string    `json:"createdObjectID,omitempty"`
		Error           *string    `json:"error,omitempty"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	id, err := TaskIDFromString(aux.ID)
	if err != nil {
		return err
	}

	var connectorID *ConnectorID
	if aux.ConnectorID != nil {
		c, err := ConnectorIDFromString(*aux.ConnectorID)
		if err != nil {
			return err
		}
		connectorID = &c
	}

	t.ID = *id
	t.ConnectorID = connectorID
	t.Status = aux.Status
	t.CreatedAt = aux.CreatedAt
	t.UpdatedAt = aux.UpdatedAt
	t.CreatedObjectID = aux.CreatedObjectID
	t.Error = func() error {
		if aux.Error == nil {
			return nil
		}

		return errors.New(*aux.Error)
	}()

	return nil
}
