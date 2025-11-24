package workflow

import (
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const PUBLISH_EVENT_BATCH_SIZE = 100

func (w Workflow) runOutboxPublisher(ctx workflow.Context) error {
	// Process a batch of pending events with no retries
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // No retries - fail immediately
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := activities.OutboxPublishPendingEvents(
		ctx,
		PUBLISH_EVENT_BATCH_SIZE,
	)
	if err != nil {
		return err
	}

	return nil
}

const RunOutboxPublisher = "OutboxPublisher"
