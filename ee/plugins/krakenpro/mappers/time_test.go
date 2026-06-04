package mappers

import (
	"math"
	"testing"
	"time"
)

func TestFloatEpochToTime(t *testing.T) {
	t.Parallel()
	// 1688019200.123456789 → 2023-06-29T08:53:20.123456789Z
	got := FloatEpochToTime(1688019200.5)
	want := time.Unix(1688019200, 500_000_000).UTC()
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestFloatEpochToTimeInvalid(t *testing.T) {
	t.Parallel()
	for _, v := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		got := FloatEpochToTime(v)
		if !got.IsZero() {
			t.Errorf("FloatEpochToTime(%v) = %v, want zero", v, got)
		}
	}
}
