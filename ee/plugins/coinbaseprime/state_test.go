package coinbaseprime

import "testing"

func TestAdvanceCursor_UsesNextWhenNonEmpty(t *testing.T) {
	t.Parallel()

	got := advanceCursor("old", "next-cursor")
	if got != "next-cursor" {
		t.Errorf("want next-cursor, got %q", got)
	}
}

func TestAdvanceCursor_KeepsOldWhenNextEmpty(t *testing.T) {
	t.Parallel()

	// End-of-pagination: Coinbase returns empty next_cursor. The plugin
	// must keep the previous cursor rather than resetting to the start of
	// history.
	got := advanceCursor("old", "")
	if got != "old" {
		t.Errorf("want old, got %q", got)
	}
}

func TestAdvanceCursor_FirstRun(t *testing.T) {
	t.Parallel()

	// First-ever call: both oldCursor and nextCursor empty → stay empty.
	got := advanceCursor("", "")
	if got != "" {
		t.Errorf("want empty cursor, got %q", got)
	}
}
