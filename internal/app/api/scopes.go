package api

const (
	scopeReadPayments  = "payments:read"
	scopeWritePayments = "payments:write"

	scopeReadConnectors  = "connectors:read"
	scopeWriteConnectors = "connectors:write"
)

var AllScopes = []string{
	scopeReadPayments,
	scopeWritePayments,

	scopeReadConnectors,
	scopeWriteConnectors,
}
