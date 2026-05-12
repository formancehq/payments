package plugins_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins"
)

// Backward-compat invariant: every existing connector that returns
// plugins.ErrUpstreamRatelimit (with errors.Is at the call-site) must keep
// working when a wrapped RateLimitedError flows through the same channel.
func TestRateLimitedErrorIsErrUpstreamRatelimit(t *testing.T) {
	err := plugins.NewRateLimitedError(5*time.Second, errors.New("429 too many requests"))
	if !errors.Is(err, plugins.ErrUpstreamRatelimit) {
		t.Fatalf("RateLimitedError must satisfy errors.Is(_, ErrUpstreamRatelimit) for backward compatibility")
	}
}

func TestRateLimitedErrorPreservesCauseInIs(t *testing.T) {
	cause := fmt.Errorf("connection reset by peer")
	err := plugins.NewRateLimitedError(0, cause)
	if !errors.Is(err, cause) {
		t.Fatalf("RateLimitedError must keep its Cause reachable via errors.Is")
	}
}

func TestRateLimitedErrorAsExposesHint(t *testing.T) {
	err := error(plugins.NewRateLimitedError(7*time.Second, nil))
	var typed *plugins.RateLimitedError
	if !errors.As(err, &typed) {
		t.Fatalf("errors.As must recover *RateLimitedError")
	}
	if typed.RetryAfter != 7*time.Second {
		t.Fatalf("RetryAfter want 7s, got %s", typed.RetryAfter)
	}
}

func TestRateLimitedErrorNegativeHintNormalizedToZero(t *testing.T) {
	err := plugins.NewRateLimitedError(-1*time.Second, nil)
	if err.RetryAfter != 0 {
		t.Fatalf("negative RetryAfter must be normalised to 0, got %s", err.RetryAfter)
	}
}

// Error() with a cause: surfaces both the canonical sentinel message and
// the underlying transport error so log triage has the original context.
func TestRateLimitedErrorMessageWithCause(t *testing.T) {
	cause := errors.New("connection reset by peer")
	got := plugins.NewRateLimitedError(5*time.Second, cause).Error()
	if !strings.Contains(got, plugins.ErrUpstreamRatelimit.Error()) {
		t.Errorf("missing rate-limit sentinel in %q", got)
	}
	if !strings.Contains(got, "connection reset by peer") {
		t.Errorf("missing cause in %q", got)
	}
}

// Error() without a cause: returns just the sentinel string (no spurious
// suffix).
func TestRateLimitedErrorMessageWithoutCause(t *testing.T) {
	got := plugins.NewRateLimitedError(0, nil).Error()
	if got != plugins.ErrUpstreamRatelimit.Error() {
		t.Errorf("Error() with no cause = %q, want %q", got, plugins.ErrUpstreamRatelimit.Error())
	}
}

// Unwrap() with no cause: just exposes the canonical sentinel so
// errors.Is(_, ErrUpstreamRatelimit) keeps working when the plugin
// surfaces a hint without a wrapped transport error.
func TestRateLimitedErrorUnwrapNoCauseStillExposesSentinel(t *testing.T) {
	err := error(plugins.NewRateLimitedError(3*time.Second, nil))
	if !errors.Is(err, plugins.ErrUpstreamRatelimit) {
		t.Fatalf("errors.Is(_, ErrUpstreamRatelimit) must hold even without a cause")
	}
}
