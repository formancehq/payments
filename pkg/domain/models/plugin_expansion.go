package models

//go:generate mockgen -source plugin_expansion.go -destination plugin_expansion_generated.go -package models

// PluginWithBootstrapOnInstall is an optional upgrade on Plugin. A plugin
// that implements it declares one or more fetch tasks that must run to
// completion (HasMore: false) as part of the install flow, before any of
// the plugin's periodic schedules are registered. The declared tasks run
// sequentially in the returned order.
//
// Dispatcher support is currently narrow: only TASK_FETCH_ACCOUNTS is
// wired through runBootstrapTask. Any other TaskType returned here will
// fail the bootstrap workflow with a non-retryable error at runtime.
// Extending the dispatcher (see runBootstrapTask in
// internal/connectors/engine/workflow/bootstrap_task.go) is required
// before declaring additional task types.
type PluginWithBootstrapOnInstall interface {
	BootstrapOnInstall() []TaskType
}

// PluginWithPayoutThrottle is an optional upgrade on Plugin. A plugin that
// implements it caps how many payout and transfer initiations the engine sends
// to the PSP per second: CreatePayout and CreateTransfer run on a dedicated
// Temporal task queue whose worker has TaskQueueActivitiesPerSecond set to the
// returned value, and those workflows route every activity except the plugin
// call itself back to the default queue. So the returned value counts calls
// that reach the PSP, not activities.
//
// The budget covers initiations only. Polling a payout's status, reversals and
// the periodic fetch tasks all run unthrottled on the default queue.
//
// Returning 0 (or dropping the interface) stops the dedicated worker from being
// started at all. Payout workflows already in flight live on that queue until
// they finish - including while sleeping until a future ScheduledAt - so drain
// them before making that change, or they will sit with no poller.
type PluginWithPayoutThrottle interface {
	PayoutsPerSecond() float64
}
