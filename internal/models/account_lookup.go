package models

import "context"

//go:generate mockgen -source account_lookup.go -destination account_lookup_generated.go -package models . AccountLookup

// AccountLookup gives a plugin read-only access to the engine's accounts
// table, scoped to the plugin's own connector. The engine injects an
// implementation via PluginWithAccountLookup; plugins that need to resolve
// wallets/accounts on demand call it from their fetch paths instead of
// keeping an in-memory side-table (which is not safe across pods).
type AccountLookup interface {
	ListAccountsByConnector(ctx context.Context) ([]PSPAccount, error)
}

// PluginWithAccountLookup is an optional upgrade on Plugin. A plugin that
// implements it signals to the engine that it wants an AccountLookup wired
// in. The engine calls UseAccountLookup once per plugin instance, before any
// activity or workflow dispatches to the plugin.
type PluginWithAccountLookup interface {
	UseAccountLookup(AccountLookup)
}

// AccountLookupFactory builds an AccountLookup scoped to a single connector.
// The engine passes a non-nil factory to the connectors manager so that the
// manager can inject a per-connector AccountLookup into every plugin it
// loads that implements PluginWithAccountLookup.
type AccountLookupFactory func(ConnectorID) AccountLookup

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
