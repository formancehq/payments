package events

import (
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type TaskMessagePayload struct {
	ID              string    `json:"id"`
	ConnectorID     *string   `json:"connectorID"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	CreatedObjectID *string   `json:"createdObjectID"`
	Error           *string   `json:"error"`
}

func (e Events) NewEventUpdatedTask(task models.Task) publish.EventMessage {
	payload := TaskMessagePayload{
		ID: task.ID.String(),
		ConnectorID: func() *string {
			if task.ConnectorID == nil {
				return nil
			}

			return pointer.For(task.ConnectorID.String())
		}(),
		Status:          string(task.Status),
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
		CreatedObjectID: task.CreatedObjectID,
		Error: func() *string {
			if task.Error == nil {
				return nil
			}

			return pointer.For(task.Error.Error())
		}(),
	}

	return publish.EventMessage{
		IdempotencyKey: task.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeUpdatedTask,
		Payload:        payload,
	}
}
