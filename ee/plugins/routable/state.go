package routable

import "time"

// pageState is the shared shape used by every fetcher in this plugin. The
// engine treats State as opaque JSON; we reuse a single struct to keep the
// "page + watermark" pattern consistent across resources.
//
// Page is the next Routable page (1-indexed) to request. LastSeenAt is the
// last status_changed_at watermark we passed to Routable; it is only set for
// resources that support the status_changed_at.gte filter (payables and
// receivables today).
type pageState struct {
	Page       int       `json:"page"`
	LastSeenAt time.Time `json:"lastSeenAt,omitempty"`
}

// nextPage returns the page number to request given the previous state.
// First-run returns 1.
func (s pageState) nextPage() int {
	if s.Page <= 0 {
		return 1
	}
	return s.Page
}
