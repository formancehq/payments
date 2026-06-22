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

// AccountLookupFactory builds an AccountLookup scoped to a single connector.
// The engine passes a non-nil factory to the connectors manager so that the
// manager can inject a per-connector AccountLookup into every plugin it
// loads that implements PluginWithAccountLookup.
type AccountLookupFactory func(ConnectorID) AccountLookup
