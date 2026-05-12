package httpwrapper

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins"
)

// Exhaustive coverage of the four header shapes the standards allow:
//   - RFC 9110 Retry-After: delta-seconds (positive, negative, zero)
//   - RFC 9110 Retry-After: HTTP-date (future, past)
//   - IETF draft RateLimit: single policy with t
//   - IETF draft RateLimit: multiple policies (smallest t wins)
//   - IETF draft RateLimit: list split across multiple headers
//
// Precedence (per IETF §6): Retry-After wins when present, else the
// smallest t across listed policies, else nil.
func TestParseRateLimitHeaders(t *testing.T) {
	now := time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		headers  http.Header
		wantWait time.Duration
		wantHas  bool
	}{
		{
			name:     "no headers",
			headers:  http.Header{},
			wantWait: 0,
			wantHas:  false,
		},
		{
			name:     "Retry-After delta-seconds",
			headers:  http.Header{"Retry-After": []string{"7"}},
			wantWait: 7 * time.Second,
			wantHas:  true,
		},
		{
			name:     "Retry-After negative delta normalised to zero",
			headers:  http.Header{"Retry-After": []string{"-1"}},
			wantWait: 0,
			wantHas:  true,
		},
		{
			name:     "Retry-After HTTP-date",
			headers:  http.Header{"Retry-After": []string{now.Add(45 * time.Second).Format(http.TimeFormat)}},
			wantWait: 45 * time.Second,
			wantHas:  true,
		},
		{
			name:     "Retry-After HTTP-date in the past clamps to zero",
			headers:  http.Header{"Retry-After": []string{now.Add(-30 * time.Second).Format(http.TimeFormat)}},
			wantWait: 0,
			wantHas:  true,
		},
		{
			name:     "RateLimit single policy with t",
			headers:  http.Header{"Ratelimit": []string{`"default";r=0;t=12`}},
			wantWait: 12 * time.Second,
			wantHas:  true,
		},
		{
			name:     "RateLimit picks smallest t across multiple policies",
			headers:  http.Header{"Ratelimit": []string{`"permin";r=0;t=15, "perhour";r=900;t=1500`}},
			wantWait: 15 * time.Second,
			wantHas:  true,
		},
		{
			name: "RateLimit split across multiple headers picks smallest",
			headers: http.Header{"Ratelimit": []string{
				`"permin";r=0;t=8`,
				`"perday";r=10000;t=86400`,
			}},
			wantWait: 8 * time.Second,
			wantHas:  true,
		},
		{
			name: "Retry-After wins over RateLimit per IETF §6",
			headers: http.Header{
				"Retry-After": []string{"60"},
				"Ratelimit":   []string{`"default";r=0;t=10`},
			},
			wantWait: 60 * time.Second,
			wantHas:  true,
		},
		{
			name:     "RateLimit without t parameter is ignored but signals presence",
			headers:  http.Header{"Ratelimit": []string{`"default";r=0`}},
			wantWait: 0,
			wantHas:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotWait, gotHas := ParseRateLimitHeaders(tc.headers, now)
			if gotWait != tc.wantWait {
				t.Errorf("wait want %s got %s", tc.wantWait, gotWait)
			}
			if gotHas != tc.wantHas {
				t.Errorf("hasSignal want %v got %v", tc.wantHas, gotHas)
			}
		})
	}
}

// classifyRateLimitResponse contract:
//   - 429 is rate-limited regardless of headers (some PSPs return bare 429).
//   - 5xx is rate-limited ONLY when the server attached a RateLimit
//     header — per IETF §6, those headers can accompany any status code
//     and their presence on a 5xx is the server explicitly signalling
//     quota/capacity rather than a generic outage.
//   - 2xx and 4xx (other than 429) are never rate-limited.
//   - parsed waits are clamped to MaxRateLimitHint to defend against
//     absurd Retry-After values (RFC 9110 §8.4 warning).
func TestClassifyRateLimitResponse(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		headers  http.Header
		want     bool
		wantWait time.Duration
	}{
		{"429 with no hint", 429, http.Header{}, true, 0},
		{"429 with Retry-After", 429, http.Header{"Retry-After": []string{"45"}}, true, 45 * time.Second},
		{"429 with RateLimit", 429, http.Header{"Ratelimit": []string{`"default";r=0;t=30`}}, true, 30 * time.Second},
		{"429 with absurd Retry-After clamped to ceiling", 429, http.Header{"Retry-After": []string{"86400"}}, true, MaxRateLimitHint},
		{"503 with RateLimit (capacity signal)", 503, http.Header{"Ratelimit": []string{`"default";r=0;t=120`}}, true, 120 * time.Second},
		{"503 with absurd RateLimit t clamped to ceiling", 503, http.Header{"Ratelimit": []string{`"default";r=0;t=999999`}}, true, MaxRateLimitHint},
		{"503 without RateLimit (generic outage)", 503, http.Header{}, false, 0},
		{"500 without RateLimit", 500, http.Header{}, false, 0},
		{"502 without RateLimit", 502, http.Header{}, false, 0},
		{"400 with RateLimit is ignored", 400, http.Header{"Ratelimit": []string{`"default";r=0;t=10`}}, false, 0},
		{"201 with RateLimit is ignored (success)", 201, http.Header{"Ratelimit": []string{`"default";r=99;t=60`}}, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tc.status, Header: tc.headers}
			got, gotWait := classifyRateLimitResponse(resp)
			if got != tc.want {
				t.Errorf("rateLimited want %v got %v", tc.want, got)
			}
			if gotWait != tc.wantWait {
				t.Errorf("retryAfter want %s got %s", tc.wantWait, gotWait)
			}
		})
	}
}

// End-to-end: rate-limit parsing is on by default. Do() upgrades a 429
// into *plugins.RateLimitedError carrying the parsed hint. Keeps the
// httpwrapper status sentinel intact via wrap-Cause so callers that
// branch on either errors.Is(err, ErrStatusCodeTooManyRequests) OR
// errors.Is(err, plugins.ErrUpstreamRatelimit) keep working.
func TestDo_Default_Wraps429AsRateLimitedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "42")
		w.Header().Set("RateLimit", `"default";r=0;t=42`)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = fmt.Fprint(w, `{"type":"https://iana.org/assignments/http-problem-types#quota-exceeded"}`)
	}))
	defer srv.Close()

	c := NewClient(&Config{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	var errBody map[string]any
	_, err := c.Do(context.Background(), req, nil, &errBody)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, plugins.ErrUpstreamRatelimit) {
		t.Errorf("expected errors.Is(_, ErrUpstreamRatelimit), got %v", err)
	}
	if !errors.Is(err, ErrStatusCodeTooManyRequests) {
		t.Errorf("expected errors.Is(_, ErrStatusCodeTooManyRequests) to still hold for backward compatibility, got %v", err)
	}
	var rl *plugins.RateLimitedError
	if !errors.As(err, &rl) {
		t.Fatalf("expected *plugins.RateLimitedError, got %T", err)
	}
	if rl.RetryAfter != 42*time.Second {
		t.Errorf("RetryAfter want 42s got %s", rl.RetryAfter)
	}
}

// Explicit opt-out: when DisableRateLimitHints=true, Do() bypasses the
// rate-limit upgrade and returns the plain status-driven sentinel — the
// escape hatch for connectors that need to keep the old behaviour.
func TestDo_DisableRateLimitHints_NoRateLimitedErrorWrapper(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "42")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewClient(&Config{DisableRateLimitHints: true})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	_, err := c.Do(context.Background(), req, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrStatusCodeTooManyRequests) {
		t.Errorf("expected ErrStatusCodeTooManyRequests, got %v", err)
	}
	if errors.Is(err, plugins.ErrUpstreamRatelimit) {
		t.Errorf("opt-out client must NOT wrap into RateLimitedError; got %v", err)
	}
}

// 503 with a RateLimit header (per IETF §6 a quota signal, not a
// generic outage) classifies as rate-limited under the default config.
func TestDo_Default_Wraps503WithRateLimitHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("RateLimit", `"capacity";r=0;t=120`)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := NewClient(&Config{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	_, err := c.Do(context.Background(), req, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, plugins.ErrUpstreamRatelimit) {
		t.Fatalf("503 with RateLimit header must classify as rate-limited; got %v", err)
	}
	var rl *plugins.RateLimitedError
	if !errors.As(err, &rl) || rl.RetryAfter != 120*time.Second {
		t.Fatalf("expected RetryAfter=120s; got %v", err)
	}
}

// Plain 5xx without rate-limit headers stays a generic server error
// (engine applies its standard backoff). Guards against the
// over-classification trap where every transient outage gets treated
// as a quota event.
func TestDo_Default_503WithoutHeaderStaysServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := NewClient(&Config{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	_, err := c.Do(context.Background(), req, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, plugins.ErrUpstreamRatelimit) {
		t.Fatalf("plain 5xx must NOT classify as rate-limited; got %v", err)
	}
	if !errors.Is(err, ErrStatusCodeServerError) {
		t.Fatalf("expected ErrStatusCodeServerError; got %v", err)
	}
}

// Absurd Retry-After values must be clamped to MaxRateLimitHint
// (RFC 9110 §8.4 warning): a misbehaving server can't park a workflow
// attempt for hours/days by replying with Retry-After: 86400.
func TestDo_Default_ClampsAbsurdRetryAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "86400")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewClient(&Config{})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	_, err := c.Do(context.Background(), req, nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var rl *plugins.RateLimitedError
	if !errors.As(err, &rl) {
		t.Fatalf("expected *plugins.RateLimitedError, got %T", err)
	}
	if rl.RetryAfter != MaxRateLimitHint {
		t.Errorf("RetryAfter want %s (clamped); got %s", MaxRateLimitHint, rl.RetryAfter)
	}
}
