package workflow

import "go.temporal.io/sdk/workflow"

const (
	versionFlagOutboxPatternEnabled  = "event_outbox_pattern_enabled"
	versionFlagRunNextTaskAsActivity = "run_next_task_as_activity"
)

func IsEventOutboxPatternEnabled(ctx workflow.Context) bool {
	version := workflow.GetVersion(ctx, versionFlagOutboxPatternEnabled, workflow.DefaultVersion, 1)
	if version > workflow.DefaultVersion {
		return true
	}
	return false
}

func IsRunNextTaskAsActivityEnabled(ctx workflow.Context) bool {
	version := workflow.GetVersion(ctx, versionFlagRunNextTaskAsActivity, workflow.DefaultVersion, 1)
	if version > workflow.DefaultVersion {
		return true
	}
	return false
}
