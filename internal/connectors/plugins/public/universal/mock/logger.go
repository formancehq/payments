package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// Structured logging helpers for the mock counterparty. Every HTTP
// request gets a short request ID propagated via context, included on
// every log line, AND echoed back in the X-Mock-Request-ID response
// header so the operator can correlate engine-side logs with mock-side
// activity.
//
// Verbosity is controlled by MOCK_LOG_LEVEL: debug | info (default) | warn | error.

type ctxKey string

const ridKey ctxKey = "rid"

func newLogger(level string) *slog.Logger {
	lvl := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}
	// Text handler is the most readable for tail -f docker logs; JSON
	// would be marginally better for log shippers but every contributor
	// reads docker logs interactively first.
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
		// Strip the time prefix when run under docker — docker already
		// prepends one, doubling it just adds noise.
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})
	return slog.New(h)
}

// shortID returns a 6-byte (12-char) hex string. Short enough to keep
// log lines compact, wide enough to dedup across a few hundred concurrent
// requests in a single session.
func shortID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "noid"
	}
	return hex.EncodeToString(b[:])
}

// reqLogger pulls the per-request logger out of context. Every middleware
// stack puts one there; falling back to the server's base logger keeps
// us from panicking on out-of-band code paths (e.g. auto-evolve ticker
// goroutine) where there is no request context.
func reqLogger(ctx context.Context, base *slog.Logger) *slog.Logger {
	if l, ok := ctx.Value(ridKey).(*slog.Logger); ok {
		return l
	}
	return base
}

// loggingMiddleware assigns a request ID, logs the inbound request +
// final status + duration, and echoes the ID back as
// X-Mock-Request-ID so the operator can correlate. Wraps every route
// EXCEPT /healthz which is intentionally silent (Docker healthcheck
// pings every 10s).
func (s *server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		rid := shortID()
		l := s.logger.With(
			"rid", rid,
			"method", r.Method,
			"path", r.URL.Path,
		)
		// Surface the most relevant request-time fields up front so
		// the operator can spot what's interesting at a glance.
		fields := []any{}
		if q := r.URL.RawQuery; q != "" {
			fields = append(fields, "query", q)
		}
		if idem := r.Header.Get("Idempotency-Key"); idem != "" {
			fields = append(fields, "idem", idem)
		}
		l.With(fields...).Debug("→ request")

		ctx := context.WithValue(r.Context(), ridKey, l)
		w.Header().Set("X-Mock-Request-ID", rid)

		ww := &statusCapture{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(ww, r.WithContext(ctx))
		l.With("status", ww.status, "ms", time.Since(start).Milliseconds()).Info("← response")
	})
}

// statusCapture is a minimal http.ResponseWriter wrapper that remembers
// the status code so the logging middleware can include it in the
// "← response" log line.
type statusCapture struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *statusCapture) WriteHeader(s int) {
	if !w.wrote {
		w.status = s
		w.wrote = true
	}
	w.ResponseWriter.WriteHeader(s)
}

func (w *statusCapture) Write(b []byte) (int, error) {
	if !w.wrote {
		w.status = http.StatusOK
		w.wrote = true
	}
	return w.ResponseWriter.Write(b)
}

// Defensive sink for any code that wants to silently log to a writer
// instead of slog (e.g. http.Server's ErrorLog).
var discardWriter io.Writer = io.Discard
