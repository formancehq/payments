package routable

import "time"

// pageState is the cursor for fetchers without a status_changed_at
// filter (settings accounts, companies). The engine treats State as
// opaque JSON; using a shared shape keeps the per-resource fetchers
// trivial. Payments use a richer paymentsState (see payments.go).
// LastCompletedAt is set by fetchers that throttle (see external_accounts.go).
type pageState struct {
	Page            int       `json:"page"`
	LastCompletedAt time.Time `json:"lastCompletedAt,omitempty"`
}

func (s pageState) nextPage() int {
	if s.Page <= 0 {
		return 1
	}
	return s.Page
}

func (s pageState) isStartOfCycle() bool { return s.Page <= 1 }
