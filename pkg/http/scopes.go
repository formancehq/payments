package http

const (
	ScopeReadPayments    = "payments:read"
	ScopeWritePayments   = "payments:write"
	ScopeReadConnectors  = "connectors:read"
	ScopeWriteConnectors = "connectors:write"
)

var AllScopes = []string{
	ScopeReadPayments,
	ScopeWritePayments,
	ScopeReadConnectors,
	ScopeWriteConnectors,
}
