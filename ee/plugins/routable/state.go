package routable

// pageState is the cursor for fetchers without a status_changed_at
// filter (settings accounts, companies). The engine treats State as
// opaque JSON; using a shared shape keeps the per-resource fetchers
// trivial. Payments use a richer paymentsState (see payments.go).
type pageState struct {
	Page int `json:"page"`
}

func (s pageState) nextPage() int {
	if s.Page <= 0 {
		return 1
	}
	return s.Page
}
