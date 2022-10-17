package api

const (
	scopeReadPayments  = "payments:read"
	scopeWritePayments = "payments:write"

	scopeReadConnectors  = "connectors:read"
	scopeWriteConnectors = "connectors:write"
)

func allScopes() []string {
	return []string{
		scopeReadPayments,
		scopeWritePayments,

		scopeReadConnectors,
		scopeWriteConnectors,
	}
}
