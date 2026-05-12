package httpwrapper

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// MaxRateLimitHint caps the wait we will honour from an upstream
// server. RFC 9110 §8.4 explicitly warns that Retry-After values can be
// excessive (accidentally or maliciously); this ceiling preserves
// Temporal's default retry behaviour as the upper bound and prevents a
// single misbehaving response from parking a workflow attempt for days.
const MaxRateLimitHint = time.Hour

// ParseRateLimitHeaders reads the wait hint per RFC 9110 §10.2.3
// (Retry-After) and draft-ietf-httpapi-ratelimit-headers-10 §4.1.2
// (RateLimit "t" parameter). Precedence per IETF §6: Retry-After wins,
// then the smallest "t" across listed policies, then zero. The bool
// distinguishes "no signal" from "0 seconds, retry now".
//
// Honoured by httpwrapper Do() by default; opt out via
// Config.DisableRateLimitHints. The parsed wait surfaces as
// *plugins.RateLimitedError, which the engine's temporalPluginErrorCheck
// feeds into Temporal's per-error NextRetryDelay.
func ParseRateLimitHeaders(h http.Header, now time.Time) (time.Duration, bool) {
	hasSignal := false

	if v := strings.TrimSpace(h.Get("Retry-After")); v != "" {
		hasSignal = true
		if d, ok := parseRetryAfter(v, now); ok {
			return d, true
		}
	}

	if values := h.Values("RateLimit"); len(values) > 0 {
		hasSignal = true
		if d, ok := parseRateLimitField(values); ok {
			return d, true
		}
	}

	return 0, hasSignal
}

// parseRetryAfter accepts RFC 9110's two shapes: delta-seconds or
// HTTP-date.
func parseRetryAfter(v string, now time.Time) (time.Duration, bool) {
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0, true
		}
		return time.Duration(secs) * time.Second, true
	}
	for _, layout := range []string{http.TimeFormat, time.RFC1123, time.RFC1123Z, time.RFC850, time.ANSIC} {
		if t, err := time.Parse(layout, v); err == nil {
			d := t.Sub(now)
			if d < 0 {
				return 0, true
			}
			return d, true
		}
	}
	return 0, false
}

// parseRateLimitField returns the smallest "t" (reset) across listed
// policies. We don't pull in a full RFC 8941 SF parser because we only
// need the "t" parameter. Accepted shapes:
//
//	RateLimit: "default";r=50;t=30
//	RateLimit: "permin";r=0;t=12, "perhour";r=900;t=1500
func parseRateLimitField(values []string) (time.Duration, bool) {
	var (
		smallest time.Duration
		found    bool
	)
	for _, raw := range values {
		for _, policy := range splitTopLevelCommas(raw) {
			for _, part := range strings.Split(policy, ";") {
				part = strings.TrimSpace(part)
				if !strings.HasPrefix(part, "t=") {
					continue
				}
				secs, err := strconv.Atoi(strings.TrimPrefix(part, "t="))
				if err != nil || secs < 0 {
					continue
				}
				d := time.Duration(secs) * time.Second
				if !found || d < smallest {
					smallest = d
					found = true
				}
			}
		}
	}
	return smallest, found
}

// splitTopLevelCommas splits on commas outside quoted strings — RFC 8941
// allows escaped quotes inside quoted strings, so a naïve strings.Split
// would mis-handle inputs like `"a,b";q=1`.
func splitTopLevelCommas(s string) []string {
	var (
		out     []string
		buf     strings.Builder
		inQuote bool
		escape  bool
	)
	for _, r := range s {
		switch {
		case escape:
			buf.WriteRune(r)
			escape = false
		case r == '\\' && inQuote:
			buf.WriteRune(r)
			escape = true
		case r == '"':
			buf.WriteRune(r)
			inQuote = !inQuote
		case r == ',' && !inQuote:
			out = append(out, strings.TrimSpace(buf.String()))
			buf.Reset()
		default:
			buf.WriteRune(r)
		}
	}
	if last := strings.TrimSpace(buf.String()); last != "" {
		out = append(out, last)
	}
	return out
}

// classifyRateLimitResponse flags whether a response should be surfaced
// as a rate-limit error. Per IETF §6, RateLimit headers can accompany any
// status code; their presence on a 5xx is the server signalling quota/
// capacity, not a generic outage. 429 is always classified as rate-limit
// regardless of headers.
func classifyRateLimitResponse(resp *http.Response) (rateLimited bool, retryAfter time.Duration) {
	if resp == nil {
		return false, 0
	}
	wait, hasSignal := ParseRateLimitHeaders(resp.Header, time.Now())
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return true, clampRateLimitHint(wait)
	case resp.StatusCode >= http.StatusInternalServerError && hasSignal:
		return true, clampRateLimitHint(wait)
	}
	return false, 0
}

// clampRateLimitHint caps a parsed hint to MaxRateLimitHint. Negative
// or zero values are passed through untouched (a zero hint means "no
// wait", which the engine then clamps against its own floor).
func clampRateLimitHint(d time.Duration) time.Duration {
	if d > MaxRateLimitHint {
		return MaxRateLimitHint
	}
	return d
}
