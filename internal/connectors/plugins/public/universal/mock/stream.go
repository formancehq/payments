package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Tunings mirror the plugin-side supervisor so test scenarios share one
// clock-skew budget; nonce TTL is generous enough to cover any plausible
// pod-restart + reconnect within tolerance.
const (
	streamSubProtocol     = "formance-universal-v1"
	streamSkewTolerance   = 5 * time.Minute
	streamNonceTTL        = 10 * time.Minute
	streamReadLimit       = 1 << 20
	streamWriteTimeout    = 10 * time.Second
	streamHandshakeBudget = 15 * time.Second
)

// streamHub fans broadcast frames out to every connected subscriber for
// a given event name. The mock keeps the hub trivial: one goroutine per
// connection, all writes serialised through that goroutine via the
// subscription's `out` channel — no contention on the hub mutex during
// the hot path.
type streamHub struct {
	logger *slog.Logger

	mu     sync.RWMutex
	subs   map[*streamSub]struct{}
	nonces *nonceCache
}

// streamSub is one connected client. `events` is the set the client
// subscribed to in its signed hello; `out` is the per-connection write
// queue. Broadcast picks `out` non-blockingly to avoid wedging the hub
// on a slow consumer.
type streamSub struct {
	events map[string]struct{}
	out    chan []byte
}

func newStreamHub(logger *slog.Logger) *streamHub {
	return &streamHub{
		logger: logger.With("component", "stream-hub"),
		subs:   map[*streamSub]struct{}{},
		nonces: newNonceCache(streamNonceTTL),
	}
}

// Broadcast pushes the JSON-encoded envelope to every subscriber that
// asked for `eventName`. Returns the number of subscribers the frame
// was queued for. Subscribers whose `out` channel is full have the
// frame dropped (counted in logs); the plugin-side supervisor accepts
// at-least-once and dedups via event id downstream.
func (h *streamHub) Broadcast(ctx context.Context, eventName string, ev map[string]any) int {
	buf, err := json.Marshal(ev)
	if err != nil {
		h.logger.Error("stream broadcast marshal failed", "event", eventName, "error", err)
		return 0
	}
	h.mu.RLock()
	targets := make([]*streamSub, 0, len(h.subs))
	for sub := range h.subs {
		if _, ok := sub.events[eventName]; ok {
			targets = append(targets, sub)
		}
	}
	h.mu.RUnlock()
	delivered := 0
	for _, sub := range targets {
		select {
		case sub.out <- buf:
			delivered++
		default:
			h.logger.Warn("stream broadcast dropped (subscriber queue full)", "event", eventName)
		}
	}
	_ = ctx // reserved for future per-frame tracing
	return delivered
}

// Subscribers returns the current connected-subscriber count. Used by
// the metrics-style logging in `handleStream` and by tests.
func (h *streamHub) Subscribers() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}

func (h *streamHub) add(sub *streamSub) {
	h.mu.Lock()
	h.subs[sub] = struct{}{}
	h.mu.Unlock()
}

func (h *streamHub) remove(sub *streamSub) {
	h.mu.Lock()
	delete(h.subs, sub)
	h.mu.Unlock()
	close(sub.out)
}

// handleStream is the HTTP handler for GET /v1/stream. Upgrades to WS,
// validates the signed hello (auth + timestamp tolerance + nonce
// freshness + signature), sends hello-ack, then spins one reader and
// one writer goroutine until the connection closes.
func (s *server) handleStream(w http.ResponseWriter, r *http.Request) {
	logger := reqLogger(r.Context(), s.logger).With("path", "/v1/stream")

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:   []string{streamSubProtocol},
		OriginPatterns: []string{"*"}, // intra-cluster only; gateway is the network boundary
	})
	if err != nil {
		logger.Warn("ws accept failed", "error", err)
		return
	}
	if got := conn.Subprotocol(); got != streamSubProtocol {
		_ = conn.Close(websocket.StatusProtocolError, "subprotocol mismatch")
		return
	}
	conn.SetReadLimit(streamReadLimit)

	hsCtx, cancel := context.WithTimeout(r.Context(), streamHandshakeBudget)
	defer cancel()

	sub, ack, reason, err := s.acceptHello(hsCtx, conn)
	if err != nil {
		_ = conn.Close(websocket.StatusPolicyViolation, reason)
		logger.Warn("ws handshake rejected", "reason", reason, "error", err)
		return
	}
	if err := writeJSONFrame(hsCtx, conn, ack); err != nil {
		_ = conn.Close(websocket.StatusInternalError, "hello-ack write failed")
		return
	}
	logger.Info("ws handshake accepted",
		"nonce", ack.Nonce,
		"accepted", ack.Accepted,
		"subscribers", s.hub.Subscribers()+1,
	)

	s.hub.add(sub)
	defer s.hub.remove(sub)
	s.serveConnection(r.Context(), conn, sub, logger)
}

// helloAckOut mirrors client.HelloAck but adds Nonce for log
// correlation (tests assert it changes between reconnects).
type helloAckOut struct {
	Type       string    `json:"type"`
	Accepted   []string  `json:"accepted"`
	ServerTime time.Time `json:"serverTime"`
	Nonce      string    `json:"-"` // log-only, not on the wire
}

// acceptHello reads the hello frame, validates every clause, and on
// success returns the subscription handle + the ack to send. The
// `reason` string surfaces in the WS close frame so the client
// reconnect logs are diagnosable.
func (s *server) acceptHello(ctx context.Context, conn *websocket.Conn) (*streamSub, *helloAckOut, string, error) {
	_, raw, err := conn.Read(ctx)
	if err != nil {
		return nil, nil, "hello read failed", err
	}
	var hello struct {
		Type      string   `json:"type"`
		APIKey    string   `json:"apiKey"`
		Timestamp string   `json:"timestamp"`
		Nonce     string   `json:"nonce"`
		Events    []string `json:"events"`
		Signature string   `json:"signature"`
	}
	if err := json.Unmarshal(raw, &hello); err != nil {
		return nil, nil, "hello decode failed", err
	}
	if hello.Type != "hello" {
		return nil, nil, "first frame must be hello", fmt.Errorf("got type %q", hello.Type)
	}
	if hello.APIKey != s.cfg.apiKey {
		return nil, nil, "apiKey rejected", fmt.Errorf("bad apiKey")
	}
	ts, err := time.Parse(time.RFC3339, hello.Timestamp)
	if err != nil {
		return nil, nil, "invalid timestamp", err
	}
	if delta := time.Since(ts); delta > streamSkewTolerance || delta < -streamSkewTolerance {
		return nil, nil, "timestamp out of tolerance", fmt.Errorf("delta %s", delta)
	}
	if hello.Nonce == "" {
		return nil, nil, "missing nonce", fmt.Errorf("empty nonce")
	}
	if !s.hub.nonces.AddIfFresh(hello.Nonce) {
		return nil, nil, "nonce already seen", fmt.Errorf("replay rejected")
	}
	expected := signHello(s.cfg.webhookSecret, hello.Timestamp, hello.Nonce, hello.Events)
	got, derr := hex.DecodeString(hello.Signature)
	want, _ := hex.DecodeString(expected)
	if derr != nil || !hmac.Equal(got, want) {
		return nil, nil, "signature mismatch", fmt.Errorf("hmac mismatch")
	}

	// Resolve subscription against the counterparty's advertised
	// stream events. `["*"]` from the client means "give me all
	// events you advertise"; otherwise intersect.
	accepted := resolveAccepted(hello.Events, s.cfg.streamEvents)
	if len(accepted) == 0 {
		return nil, nil, "no overlap with advertised stream events", fmt.Errorf("empty intersection")
	}

	sub := &streamSub{
		events: make(map[string]struct{}, len(accepted)),
		out:    make(chan []byte, 64),
	}
	for _, e := range accepted {
		sub.events[e] = struct{}{}
	}
	return sub, &helloAckOut{
		Type:       "hello-ack",
		Accepted:   accepted,
		ServerTime: time.Now().UTC(),
		Nonce:      hello.Nonce,
	}, "", nil
}

// serveConnection runs reader (consumes pings, ignores everything else)
// and writer (drains sub.out into the socket) until either side errors
// or the context is cancelled.
func (s *server) serveConnection(ctx context.Context, conn *websocket.Conn, sub *streamSub, logger *slog.Logger) {
	defer func() { _ = conn.Close(websocket.StatusGoingAway, "server closing") }()

	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		for {
			_, raw, err := conn.Read(connCtx)
			if err != nil {
				cancel()
				return
			}
			var typed struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(raw, &typed); err != nil {
				continue
			}
			if typed.Type == "ping" {
				_ = writeJSONFrame(connCtx, conn, map[string]string{"type": "pong"})
			}
		}
	}()

	for {
		select {
		case <-connCtx.Done():
			logger.Info("ws connection closed")
			return
		case buf, ok := <-sub.out:
			if !ok {
				return
			}
			writeCtx, c := context.WithTimeout(connCtx, streamWriteTimeout)
			err := conn.Write(writeCtx, websocket.MessageText, buf)
			c()
			if err != nil {
				logger.Warn("ws write failed, closing", "error", err)
				return
			}
		}
	}
}

func writeJSONFrame(ctx context.Context, conn *websocket.Conn, v any) error {
	buf, err := json.Marshal(v)
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(ctx, streamWriteTimeout)
	defer cancel()
	return conn.Write(writeCtx, websocket.MessageText, buf)
}

// signHello mirrors client.SignHello on the plugin side. One signing
// primitive across both sides of the contract — refactoring either
// requires both to move in lockstep.
func signHello(secret, timestamp, nonce string, events []string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write([]byte(nonce))
	mac.Write([]byte("."))
	evBuf, _ := json.Marshal(events)
	mac.Write(evBuf)
	return hex.EncodeToString(mac.Sum(nil))
}

// resolveAccepted intersects the client's requested event list with the
// counterparty's advertised stream events. The `["*"]` sentinel on
// either side expands to the other side's list.
func resolveAccepted(requested, advertised []string) []string {
	wildcard := func(xs []string) bool { return len(xs) == 1 && xs[0] == "*" }
	switch {
	case wildcard(advertised):
		// Counterparty publishes everything — accept whatever the
		// client asked for (other than its own "*").
		if wildcard(requested) {
			return nil // both wildcards is meaningless; force explicit
		}
		out := append([]string(nil), requested...)
		return out
	case wildcard(requested):
		return append([]string(nil), advertised...)
	}
	adv := make(map[string]struct{}, len(advertised))
	for _, a := range advertised {
		adv[a] = struct{}{}
	}
	out := make([]string, 0, len(requested))
	for _, r := range requested {
		if _, ok := adv[r]; ok {
			out = append(out, r)
		}
	}
	// Strip whitespace defensively — operator-supplied env vars
	// have a habit of bringing trailing spaces.
	for i, v := range out {
		out[i] = strings.TrimSpace(v)
	}
	return out
}

// nonceCache is a TTL-bounded "have I seen this nonce?" set. Implemented
// inline (instead of pulling a cache lib) because the access pattern is
// trivial: one Add per WS connect, expiry by wall-clock.
type nonceCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[string]time.Time
}

func newNonceCache(ttl time.Duration) *nonceCache {
	c := &nonceCache{ttl: ttl, items: map[string]time.Time{}}
	go c.janitor()
	return c
}

// AddIfFresh returns true if the nonce had not been seen within the TTL
// (and records it); false if it is a replay.
func (c *nonceCache) AddIfFresh(nonce string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	if exp, ok := c.items[nonce]; ok && now.Before(exp) {
		return false
	}
	c.items[nonce] = now.Add(c.ttl)
	return true
}

func (c *nonceCache) janitor() {
	t := time.NewTicker(c.ttl / 2)
	defer t.Stop()
	for now := range t.C {
		c.mu.Lock()
		for k, exp := range c.items {
			if now.After(exp) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
