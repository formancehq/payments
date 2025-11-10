package workflow

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const StartToCloseTimeoutMinutes = 5

func infiniteRetryContext(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: StartToCloseTimeoutMinutes * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Second,
			BackoffCoefficient:     2,
			MaximumInterval:        100 * time.Second,
			NonRetryableErrorTypes: []string{},
		},
	})
}
