package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormatToTransactionDate(t *testing.T) {
	t.Parallel()

	utc := time.FixedZone("", 0)

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
			name: "milliseconds round up to the next second",
			in:   time.Date(2026, 6, 18, 13, 54, 29, 18_000_000, utc),
			want: "2026-06-18T13:54:30+0000",
		},
		{
			name: "sub-millisecond also rounds up",
			in:   time.Date(2026, 6, 18, 13, 54, 29, 1, utc),
			want: "2026-06-18T13:54:30+0000",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, formatToTransactionDate(tc.in))
		})
	}
}
