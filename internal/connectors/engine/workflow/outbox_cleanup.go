package workflow

import (
	"time"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func (w Workflow) runOutboxCleanup(ctx workflow.Context) error {
	// Process cleanup with 5 minute timeout since deletion can take longer
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // No retries - fail immediately
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	err := activities.OutboxDeleteOldProcessedEvents(ctx)
	if err != nil {
		return err
	}

	return nil
}

const RunOutboxCleanup = "OutboxCleanup"
