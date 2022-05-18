package http

const (
	ScopeReadPayments  = "payments:read"
	ScopeWritePayments = "payments:write"
)

var AllScopes = []string{
	ScopeReadPayments,
	ScopeWritePayments,
}
