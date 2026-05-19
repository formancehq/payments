package models

//go:generate mockgen -source plugin_expansion.go -destination plugin_expansion_generated.go -package models

// PluginWithAccountLookup is an optional upgrade on Plugin. A plugin that
// implements it signals to the engine that it wants an AccountLookup wired
// in. The engine calls UseAccountLookup once per plugin instance, before any
// activity or workflow dispatches to the plugin.
type PluginWithAccountLookup interface {
	UseAccountLookup(AccountLookup)
}

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
// implements it signals to the engine that CreatePayout and CreateTransfer
// workflows should run on a dedicated Temporal task queue whose worker has
// TaskQueueActivitiesPerSecond set to the returned value.
type PluginWithPayoutThrottle interface {
	PayoutsPerSecond() float64
}
