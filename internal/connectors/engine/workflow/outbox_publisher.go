package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"go.temporal.io/sdk/workflow"
)

const PUBLISH_EVENT_BATCH_SIZE = 100

func (w Workflow) runOutboxPublisher(ctx workflow.Context) error {
	// Process a batch of pending events

	err := activities.OutboxPublishPendingEvents(
		infiniteRetryContext(ctx),
		PUBLISH_EVENT_BATCH_SIZE,
	)
	if err != nil {
		return err
	}

	return nil
}

const RunOutboxPublisher = "OutboxPublisher"
