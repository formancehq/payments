package client

type TransactionsParams struct {
	Offset         int    // default 0, max 200000
	Limit          int    // default 100, max 1000
	Sort           string // "asc" or "desc" (default: "desc")
	SinceTimestamp int64  // seconds since epoch, optional, max 30 days old (server-side rule)
	UntilTimestamp int64  // seconds since epoch, optional, max 30 days old (server-side rule)
	SinceID        string // optional; if set, limit is forced to 1000
}
