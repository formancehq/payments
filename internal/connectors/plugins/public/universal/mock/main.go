// Package main is a self-contained reference implementation of the Universal
// Connector counterparty contract. It exists so contributors can run the
// full plugin install + Temporal workflow against a known-good fixture
// without standing up a real PSP.
//
// Today's deliverable is intentionally minimal: single-tenant, in-memory,
// stdlib-only. The README outlines the planned multi-tenant evolution.
//
// Run:   go run ./internal/connectors/plugins/public/universal/mock
// Or via Docker (mock/Dockerfile).
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	cfg := loadConfig()
	logger := newLogger(cfg.logLevel)
	st := newStore(cfg, logger)
	srv := newServer(cfg, st, logger)

	if cfg.evolveInterval > 0 {
		startAutoEvolve(srv, cfg.evolveInterval, cfg.evolveBatch)
	}

	httpSrv := &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ErrorLog:          log.New(discardWriter, "", 0), // route net/http internal errors through slog instead
	}
	logger.Info("mock universal counterparty starting",
		"port", cfg.port,
		"capabilities", strings.Join(cfg.capabilities, ","),
		"signature_scheme", cfg.webhookSignature,
		"event_stream", cfg.eventStream,
		"stream_events", strings.Join(cfg.streamEvents, ","),
		"evolve_on_poll", cfg.evolveOnPoll,
		"evolve_batch", cfg.evolveBatch,
		"auto_evolve_interval", cfg.evolveInterval.String(),
		"log_level", cfg.logLevel,
	)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// nopLogger / startup convenience — keep slog in scope so the import is
// always referenced even on stripped builds.
var _ = slog.Default

// startAutoEvolve spawns a goroutine that calls server.evolveAndDeliver
// every `interval` so the dataset drifts forward without anyone hitting
// /_admin/evolve. Going through evolveAndDeliver (rather than the bare
// store.EvolveSteps) means the wall-clock ticker also auto-emits
// matching webhook events to any active subscription — closing the
// loop in webhook-mode demos. The goroutine lives for the process
// lifetime — no shutdown ceremony because the binary is meant to be
// killed via SIGTERM. Once every lane is drained it becomes a no-op.
func startAutoEvolve(srv *server, interval time.Duration, batch int) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		srv.logger.Info("auto-evolve ticker started", "interval", interval.String(), "batch", batch)
		for range ticker.C {
			if advanced, delivered := srv.evolveAndDeliver(context.Background(), batch); advanced > 0 {
				srv.logger.Info("auto-evolve tick", "advanced", advanced, "webhooks_delivered", delivered)
			}
		}
	}()
}

// mockConfig is read once at startup. Everything is overridable via env vars
// so smoke tests in CI can pin behaviour deterministically.
//
// Evolution model (the mechanism that advances records through their
// adjustment lanes — see store.go):
//
//   - evolveOnPoll (default ON) ties evolution to actual poll requests
//     on /v1/{accounts,external-accounts,payments,orders,conversions,others}.
//     Each poll advances `evolveBatch` records IF no webhook subscriptions
//     are active. With the engine's minimum polling period of 20 minutes,
//     this matches the natural cadence of state changes a real PSP would
//     show on each poll.
//
//   - When the engine has registered webhooks (CREATE_WEBHOOKS one-shot ran
//     at install), poll-driven evolution stops — the engine relies on
//     webhook pushes for state propagation. /_admin/evolve and
//     /_admin/trigger-webhook remain as manual hooks for tests + demos.
//
//   - evolveInterval > 0 (opt-in) layers an additional wall-clock ticker
//     on top — useful for pure-demo runs where no real engine is polling.
type mockConfig struct {
	port             string
	apiKey           string
	webhookSecret    string
	webhookSignature string   // "hmac-sha256" | "none"
	capabilities     []string // empty ⇒ full superset

	// eventStream advertises real-time push over WebSocket. Counterparty
	// declares "" (off) or "wss" at /v1/capabilities; clients opt in at
	// install. streamEvents lists what the counterparty publishes on
	// the stream (["*"] = "everything I publish on webhooks").
	eventStream  string
	streamEvents []string

	evolveOnPoll   bool
	evolveInterval time.Duration
	evolveBatch    int

	logLevel string // "debug" | "info" | "warn" | "error"
}

func loadConfig() mockConfig {
	c := mockConfig{
		port:             envOr("MOCK_PORT", "8080"),
		apiKey:           envOr("MOCK_API_KEY", "dev-key"),
		webhookSecret:    envOr("MOCK_WEBHOOK_SECRET", "dev-secret"),
		webhookSignature: envOr("MOCK_WEBHOOK_SIGNATURE", "hmac-sha256"),
		eventStream:      envOr("MOCK_EVENT_STREAM", ""),
		evolveOnPoll:     envOr("MOCK_EVOLVE_ON_POLL", "true") != "false",
		evolveInterval:   parseDuration(envOr("MOCK_AUTO_EVOLVE_INTERVAL", "0")),
		evolveBatch:      parseInt(envOr("MOCK_AUTO_EVOLVE_BATCH", "10"), 10),
		logLevel:         envOr("MOCK_LOG_LEVEL", "info"),
	}
	if raw := os.Getenv("MOCK_CAPABILITIES"); raw != "" {
		c.capabilities = strings.Split(raw, ",")
	} else {
		c.capabilities = defaultCapabilities()
	}
	if raw := os.Getenv("MOCK_STREAM_EVENTS"); raw != "" {
		c.streamEvents = strings.Split(raw, ",")
	} else if c.eventStream == "wss" {
		// Default: advertise the wildcard so any plugin opting into the
		// stream gets every webhook-routable event without per-deploy
		// configuration. Operators can pin the explicit list via env
		// when they want to test a narrower subscription.
		c.streamEvents = []string{"*"}
	}
	return c
}

func parseDuration(s string) time.Duration {
	if s == "" || s == "0" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("invalid duration %q, falling back to 5s: %v", s, err)
		return 5 * time.Second
	}
	return d
}

func parseInt(s string, def int) int {
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return def
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// defaultCapabilities mirrors the static superset in capabilities.go on the
// plugin side — the mock by default opts into everything so the install
// flow exercises the full Temporal workflow tree out of the box.
func defaultCapabilities() []string {
	return []string{
		"FETCH_ACCOUNTS",
		"FETCH_BALANCES",
		"FETCH_EXTERNAL_ACCOUNTS",
		"FETCH_PAYMENTS",
		"FETCH_OTHERS",
		"FETCH_ORDERS",
		"FETCH_CONVERSIONS",
		"CREATE_WEBHOOKS",
		"TRANSLATE_WEBHOOKS",
		"CREATE_BANK_ACCOUNT",
		"CREATE_TRANSFER",
		"CREATE_PAYOUT",
	}
}
