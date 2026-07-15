package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormatToPostedDate(t *testing.T) {
	t.Parallel()

	utc := time.FixedZone("", 0)

	// Modulr's toPostedDate filter is whole-second and inclusive, so truncating a
	// millisecond ceiling down to its second still returns the ceiling transaction.
	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{
			name: "whole second is left unchanged",
			in:   time.Date(2026, 6, 18, 13, 54, 29, 0, utc),
			want: "2026-06-18T13:54:29+0000",
		},
		{
			name: "milliseconds are truncated down to the second",
			in:   time.Date(2026, 6, 18, 13, 54, 29, 18_000_000, utc),
			want: "2026-06-18T13:54:29+0000",
		},
		{
			name: "sub-millisecond is also truncated down",
			in:   time.Date(2026, 6, 18, 13, 54, 29, 1, utc),
			want: "2026-06-18T13:54:29+0000",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, formatToPostedDate(tc.in))
		})
	}
}
