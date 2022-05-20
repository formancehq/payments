package bridge

const (
	ScopeReadConnectors  = "connectors:read"
	ScopeWriteConnectors = "connectors:write"
)

var AllScopes = []string{
	ScopeReadConnectors,
	ScopeWriteConnectors,
}
