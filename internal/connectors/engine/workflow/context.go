package workflow

import (
	"time"

	"github.com/formancehq/payments/pkg/domain/models"
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

// pspRetryContext pins a single activity to the queue the workflow itself was
// started on, keeping the standard infinite-retry policy. For a connector
// implementing models.PluginWithPayoutThrottle that queue is rate limited, so
// only the activities that actually reach the PSP may run there; callers route
// the rest to the default queue.
//
// workflow.GetInfo reads WorkflowInfo, which workflow.WithTaskQueue does not
// touch, so this still resolves the workflow's original queue after the caller
// has rebound ctx to the default one.
//
// This holds only because payout workflows are started exclusively via
// engine.getPayoutTaskQueue. Starting one on some other queue would pin the PSP
// call to that queue too.
func pspRetryContext(ctx workflow.Context) workflow.Context {
	return workflow.WithTaskQueue(infiniteRetryContext(ctx), workflow.GetInfo(ctx).TaskQueueName)
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
