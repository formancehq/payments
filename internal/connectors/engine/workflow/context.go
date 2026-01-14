package workflow

import (
	"time"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func infiniteRetryContext(ctx workflow.Context) workflow.Context {
	return infiniteRetryWithCustomStartToCloseAndHeartbeatContext(
		ctx,
		models.ActivityStartToCloseTimeoutMinutesDefault*time.Minute,
		time.Duration(0),
	)
}

func infiniteRetryWithLongTimeoutContext(ctx workflow.Context) workflow.Context {
	return infiniteRetryWithCustomStartToCloseAndHeartbeatContext(
		ctx,
		models.ActivityStartToCloseTimeoutMinutesLong*time.Minute,
		time.Duration(0),
	)
}
func infiniteRetryWithCustomStartToCloseAndHeartbeatContext(ctx workflow.Context, startToCloseTimeout, heartbeatTimeout time.Duration) workflow.Context {
	ao := workflow.ActivityOptions{
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Second,
			BackoffCoefficient:     2,
			MaximumInterval:        100 * time.Second,
			NonRetryableErrorTypes: []string{},
		},
	}
	if startToCloseTimeout != time.Duration(0) {
		ao.StartToCloseTimeout = startToCloseTimeout
	}
	if heartbeatTimeout != time.Duration(0) {
		ao.HeartbeatTimeout = heartbeatTimeout
	}

	return workflow.WithActivityOptions(ctx, ao)
}
