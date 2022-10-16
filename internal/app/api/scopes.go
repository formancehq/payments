package api

const (
	scopeReadPayments  = "payments:read"
	scopeWritePayments = "payments:write"

	scopeReadConnectors  = "connectors:read"
	scopeWriteConnectors = "connectors:write"
)

var allScopes = []string{
	scopeReadPayments,
	scopeWritePayments,

	scopeReadConnectors,
	scopeWriteConnectors,
}
