package workflow

import "go.temporal.io/sdk/workflow"

const (
	versionFlagOutboxPatternEnabled              = "event_outbox_pattern_enabled"
	versionFlagRunNextTaskAsActivity             = "run_next_task_as_activity"
	versionFlagPaymentInitiationUpdateAsActivity = "storage_payment_initiation_update_as_activity"
)

func IsEventOutboxPatternEnabled(ctx workflow.Context) bool {
	version := workflow.GetVersion(ctx, versionFlagOutboxPatternEnabled, workflow.DefaultVersion, 1)
	return version > workflow.DefaultVersion
}

func IsRunNextTaskAsActivityEnabled(ctx workflow.Context) bool {
	version := workflow.GetVersion(ctx, versionFlagRunNextTaskAsActivity, workflow.DefaultVersion, 1)
	return version > workflow.DefaultVersion
}

func IsPaymentInitiationUpdateAsActivityEnabled(ctx workflow.Context) bool {
	version := workflow.GetVersion(ctx, versionFlagPaymentInitiationUpdateAsActivity, workflow.DefaultVersion, 1)
	return version > workflow.DefaultVersion
}
