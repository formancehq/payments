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

// noRetryContext creates an activity context that does NOT retry on failure.
// This is critical for FOK (Fill-Or-Kill) and IOC (Immediate-Or-Cancel) orders
// because retrying these would create duplicate orders on the exchange.
func noRetryContext(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: models.ActivityStartToCloseTimeoutMinutesDefault * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // Single attempt only - no retries
		},
	})
}

// gtdRetryContext creates an activity context that retries until the expiration time.
// Used for GTD (Good-Till-Date) orders.
func gtdRetryContext(ctx workflow.Context, expiresAt time.Time) workflow.Context {
	now := workflow.Now(ctx)
	if expiresAt.Before(now) {
		// Already expired, don't retry
		return noRetryContext(ctx)
	}

	timeout := expiresAt.Sub(now)
	// Cap at a reasonable maximum (e.g., 24 hours)
	maxTimeout := 24 * time.Hour
	if timeout > maxTimeout {
		timeout = maxTimeout
	}

	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: timeout,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Second,
			BackoffCoefficient:     2,
			MaximumInterval:        100 * time.Second,
			NonRetryableErrorTypes: []string{},
		},
	})
}
