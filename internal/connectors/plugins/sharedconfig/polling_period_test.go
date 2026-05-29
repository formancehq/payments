package sharedconfig_test

import (
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
)

func TestSetAndGetPollingPeriodDefaults(t *testing.T) {
	// sequential — mutates package-level atomics
	sharedconfig.SetPollingPeriodDefaults(30*time.Minute, 20*time.Minute)
	if got := sharedconfig.GetDefaultPollingPeriod(); got != 30*time.Minute {
		t.Errorf("GetDefaultPollingPeriod() = %v, want %v", got, 30*time.Minute)
	}
	if got := sharedconfig.GetMinimumPollingPeriod(); got != 20*time.Minute {
		t.Errorf("GetMinimumPollingPeriod() = %v, want %v", got, 20*time.Minute)
	}

	// overwrite with different values
	sharedconfig.SetPollingPeriodDefaults(45*time.Minute, 10*time.Minute)
	if got := sharedconfig.GetDefaultPollingPeriod(); got != 45*time.Minute {
		t.Errorf("GetDefaultPollingPeriod() = %v, want %v", got, 45*time.Minute)
	}
	if got := sharedconfig.GetMinimumPollingPeriod(); got != 10*time.Minute {
		t.Errorf("GetMinimumPollingPeriod() = %v, want %v", got, 10*time.Minute)
	}
}

func TestParsePollingPeriod(t *testing.T) {
	cases := []struct {
		raw            string
		def, min, want time.Duration
		wantErr        bool
	}{
		{"", 30 * time.Minute, 20 * time.Minute, 30 * time.Minute, false},
		{"15m", 30 * time.Minute, 20 * time.Minute, 20 * time.Minute, false},
		{"45m", 30 * time.Minute, 20 * time.Minute, 45 * time.Minute, false},
		{"not-a-duration", 30 * time.Minute, 20 * time.Minute, 0, true},
	}
	for _, c := range cases {
		got, err := sharedconfig.NewPollingPeriod(c.raw, c.def, c.min)
		if c.wantErr && err == nil {
			t.Fatalf("expected error for value %s", c.raw)
		}
		if !c.wantErr && err != nil {
			t.Fatalf("unexpected error for value %s: %v", c.raw, err)
		}
		if !c.wantErr && got.Duration() != c.want {
			t.Fatalf("unexpected result for value %s, got %v, want %v", c.raw, got, c.want)
		}
	}
}
