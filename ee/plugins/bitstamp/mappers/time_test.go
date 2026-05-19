package mappers

import (
	"testing"
	"time"
)

func TestParseBitstampTime(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "microseconds",
			input: "2025-09-25 14:42:59.894846",
			want:  time.Date(2025, 9, 25, 14, 42, 59, 894846000, time.UTC),
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "wrong layout",
			input:   "2025-09-25T14:42:59Z",
			wantErr: true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseBitstampTime(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err: got %v wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && !got.Equal(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestBitstampGenesisIsStable(t *testing.T) {
	t.Parallel()
	// Sentinel must not drift — downstream systems rely on it being
	// the Bitstamp exchange launch date, not a moving target.
	want := time.Date(2011, 8, 2, 0, 0, 0, 0, time.UTC)
	if !BitstampGenesis.Equal(want) {
		t.Fatalf("BitstampGenesis drifted: got %v, want %v", BitstampGenesis, want)
	}
}
