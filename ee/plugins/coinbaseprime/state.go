package coinbaseprime

// incrementalState persists the opaque Coinbase Prime pagination cursor
// between fetch cycles. On each call the plugin sends Cursor as the
// `cursor` query param; Coinbase returns records strictly after it.
//
// First-run state is the zero value (empty Cursor) — the client omits the
// query param and Coinbase returns the oldest records first under the
// pinned `sort_direction=ASC`.
//
// The cursor is treated as opaque: no assumption about its format or
// lifetime is made. It is persisted verbatim and updated only when
// Coinbase returns a non-empty `next_cursor`. Coinbase returns an empty
// next_cursor at end-of-pagination; we keep the previous cursor in that
// case so the marker does not reset to the start of history. The cost is
// re-fetching the last page each cycle — the framework dedupes emitted
// records, so this is cheap and safe.
type incrementalState struct {
	Cursor string `json:"cursor"`
}

// advanceCursor returns the cursor value to persist after a page
// response. If Coinbase returned a non-empty next_cursor, use it;
// otherwise keep the previous cursor so end-of-pagination does not wipe
// the marker.
func advanceCursor(oldCursor, nextCursor string) string {
	if nextCursor != "" {
		return nextCursor
	}
	return oldCursor
}
