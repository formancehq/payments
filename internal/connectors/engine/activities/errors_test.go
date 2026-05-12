package activities

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"go.temporal.io/sdk/temporal"
)

// Engine fallback contract: a plain ErrUpstreamRatelimit (no hint) must
// still translate to a Temporal RATE_LIMITED ApplicationError with the
// configured static NextRetryDelay. Pre-existing connectors rely on this.
func TestTemporalPluginError_RateLimited_FallsBackToStaticDelay(t *testing.T) {
	a := Activities{rateLimitingRetryDelay: 30 * time.Second}
	err := a.temporalPluginError(context.Background(), plugins.ErrUpstreamRatelimit)

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *temporal.ApplicationError, got %T", err)
	}
	if appErr.Type() != ErrTypeRateLimited {
		t.Fatalf("type want %q got %q", ErrTypeRateLimited, appErr.Type())
	}
	if got := nextRetryDelay(appErr); got != 30*time.Second {
		t.Fatalf("NextRetryDelay want 30s got %s", got)
	}
}

// Honor-the-hint contract: when the plugin surfaces a RateLimitedError
// with a RetryAfter longer than the engine floor, the per-error
// NextRetryDelay must use the upstream hint.
func TestTemporalPluginError_RateLimited_HintBeatsStaticDelay(t *testing.T) {
	a := Activities{rateLimitingRetryDelay: 5 * time.Second}
	hint := 90 * time.Second
	err := a.temporalPluginError(context.Background(), plugins.NewRateLimitedError(hint, nil))

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *temporal.ApplicationError, got %T", err)
	}
	if appErr.Type() != ErrTypeRateLimited {
		t.Fatalf("type want %q got %q", ErrTypeRateLimited, appErr.Type())
	}
	if got := nextRetryDelay(appErr); got != hint {
		t.Fatalf("NextRetryDelay want %s (hint) got %s", hint, got)
	}
}

// Floor contract: a hint shorter than the engine's static floor must NOT
// pull the next-retry-delay below the floor. Operators rely on the floor
// to throttle a misconfigured PSP that sets a hint of "0s".
func TestTemporalPluginError_RateLimited_HintBelowFloorClamped(t *testing.T) {
	a := Activities{rateLimitingRetryDelay: 30 * time.Second}
	err := a.temporalPluginError(context.Background(), plugins.NewRateLimitedError(2*time.Second, nil))

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *temporal.ApplicationError, got %T", err)
	}
	if got := nextRetryDelay(appErr); got != 30*time.Second {
		t.Fatalf("NextRetryDelay must clamp to floor of 30s, got %s", got)
	}
}

// nextRetryDelay extracts the per-error delay Temporal stamped on an
// ApplicationError. The SDK exposes it directly; this helper centralises
// the call to keep the assertions readable.
func nextRetryDelay(appErr *temporal.ApplicationError) time.Duration {
	return appErr.NextRetryDelay()
}
