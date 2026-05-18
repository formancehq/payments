package client

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// StreamSubProtocol is the WS subprotocol the universal contract reserves.
// Counterparties MUST select this exact value on handshake; the client
// fails the dial otherwise.
const StreamSubProtocol = "formance-universal-v1"

// HelloFrame is the signed connect / reconnect envelope. The signature
// is HMAC-SHA256 over "<timestamp>.<nonce>.<eventsJSON>" using the
// connector's WebhookSharedSecret. Counterparties MUST reject:
//   - apiKey mismatch
//   - timestamp outside ±5min skew (matches webhook tolerance)
//   - nonce seen in the last 10min
//   - signature mismatch (constant-time compare)
//
// See contract/data-model.md "Reconnect handshake authentication".
type HelloFrame struct {
	Type      string   `json:"type"` // always "hello"
	APIKey    string   `json:"apiKey"`
	Timestamp string   `json:"timestamp"`
	Nonce     string   `json:"nonce"`
	Events    []string `json:"events"`
	Signature string   `json:"signature"`
}

// HelloAck is the server response acknowledging the handshake. Accepted
// is the subset of HelloFrame.Events the counterparty will actually push
// (intersection of what the client asked for and what /v1/capabilities
// declared). ServerTime lets the client correct clock skew before the
// next reconnect.
type HelloAck struct {
	Type       string    `json:"type"` // always "hello-ack"
	Accepted   []string  `json:"accepted"`
	ServerTime time.Time `json:"serverTime"`
}

// PingFrame / PongFrame implement application-layer keepalive. WS-level
// ping/pong is also used by coder/websocket under the hood; these
// app-level frames make end-to-end liveness debuggable from the same
// JSON log line as everything else.
type PingFrame struct {
	Type string `json:"type"` // "ping"
}

type PongFrame struct {
	Type string `json:"type"` // "pong"
}

// SignHello computes the canonical signature for a HelloFrame. Exposed
// so the mock server can recompute it for validation and tests can
// produce signed frames without duplicating the canonicalisation logic.
func SignHello(secret, timestamp, nonce string, events []string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write([]byte(nonce))
	mac.Write([]byte("."))
	mac.Write(canonicalEventsBytes(events))
	return hex.EncodeToString(mac.Sum(nil))
}

// canonicalEventsBytes serialises the events list deterministically.
// json.Marshal already sorts map keys but is non-deterministic for
// slices only insofar as encoding choices; for a flat []string we get
// "[\"a\",\"b\"]" reliably across Go versions. Keep this in one place
// so client and server agree byte-for-byte.
func canonicalEventsBytes(events []string) []byte {
	out, _ := json.Marshal(events)
	return out
}

// NewNonce returns a 128-bit random hex nonce. crypto/rand failure
// is surfaced as an error so the dial loop can apply backoff instead
// of panicking; recovery is the same as for any other transient
// failure on the supervisor's hot path.
func NewNonce() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

// StreamClient is the per-connection handle the supervisor drives.
// Read blocks until the next frame or ctx cancellation; Close sends a
// clean WS close (1001 by default) so the counterparty marks the
// subscription as gracefully disconnected.
type StreamClient struct {
	conn *websocket.Conn
	// readLimit caps the size of one frame to defend against a
	// misbehaving / malicious counterparty. 1 MiB is generous for
	// WebhookEvent payloads (typical < 4 KiB).
	readLimit int64
}

// StreamDialConfig captures every dial-time parameter. Endpoint is the
// wss:// (or ws://) URL pointing at the counterparty's /v1/stream path.
// APIKey is the same bearer the REST client uses; Secret is the
// WebhookSharedSecret (also reused for HTTP webhook verification).
// Events is the resolved subscribe list (already intersected with the
// engine's routable events).
type StreamDialConfig struct {
	Endpoint string
	APIKey   string
	Secret   string
	Events   []string
}

// DialStream performs the HTTP Upgrade and signed handshake. Returns a
// connected StreamClient and the server's HelloAck. Any failure (TLS,
// upgrade, subprotocol mismatch, hello rejected) returns an error and
// no client; the supervisor loop applies its backoff.
func DialStream(ctx context.Context, cfg StreamDialConfig) (*StreamClient, *HelloAck, error) {
	if cfg.Endpoint == "" {
		return nil, nil, fmt.Errorf("stream endpoint required")
	}
	if cfg.Secret == "" {
		return nil, nil, fmt.Errorf("stream WebhookSharedSecret required for signed handshake")
	}
	if len(cfg.Events) == 0 {
		return nil, nil, fmt.Errorf("stream events list must be non-empty")
	}

	conn, _, err := websocket.Dial(ctx, cfg.Endpoint, &websocket.DialOptions{
		Subprotocols: []string{StreamSubProtocol},
		HTTPHeader:   http.Header{"Authorization": {"Bearer " + cfg.APIKey}},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("ws dial %s: %w", cfg.Endpoint, err)
	}
	if got := conn.Subprotocol(); got != StreamSubProtocol {
		_ = conn.Close(websocket.StatusProtocolError, "subprotocol mismatch")
		return nil, nil, fmt.Errorf("ws subprotocol %q, want %q", got, StreamSubProtocol)
	}
	conn.SetReadLimit(1 << 20)

	c := &StreamClient{conn: conn, readLimit: 1 << 20}

	nonce, err := NewNonce()
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, "nonce generation failed")
		return nil, nil, fmt.Errorf("ws hello: %w", err)
	}
	hello := HelloFrame{
		Type:      "hello",
		APIKey:    cfg.APIKey,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Nonce:     nonce,
		Events:    cfg.Events,
	}
	hello.Signature = SignHello(cfg.Secret, hello.Timestamp, hello.Nonce, hello.Events)

	if err := c.writeJSON(ctx, hello); err != nil {
		_ = conn.Close(websocket.StatusInternalError, "hello write failed")
		return nil, nil, fmt.Errorf("ws hello write: %w", err)
	}

	var ack HelloAck
	if err := c.readJSON(ctx, &ack); err != nil {
		_ = conn.Close(websocket.StatusInternalError, "hello-ack read failed")
		return nil, nil, fmt.Errorf("ws hello-ack read: %w", err)
	}
	if ack.Type != "hello-ack" {
		_ = conn.Close(websocket.StatusProtocolError, "unexpected first frame")
		return nil, nil, fmt.Errorf("ws expected hello-ack, got %q", ack.Type)
	}
	return c, &ack, nil
}

// Read pulls the next decoded frame from the wire. Pong frames are
// transparently consumed so the supervisor's read loop never sees them
// — it gets WebhookEvents (the data plane) only.
func (c *StreamClient) Read(ctx context.Context) (WebhookEvent, error) {
	for {
		raw, err := c.readBytes(ctx)
		if err != nil {
			return WebhookEvent{}, err
		}
		var typed struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &typed); err != nil {
			return WebhookEvent{}, fmt.Errorf("decode frame envelope: %w", err)
		}
		if typed.Type == "pong" {
			continue
		}
		var ev WebhookEvent
		if err := json.Unmarshal(raw, &ev); err != nil {
			return WebhookEvent{}, fmt.Errorf("decode WebhookEvent: %w", err)
		}
		if ev.ID == "" || ev.Type == "" {
			return WebhookEvent{}, fmt.Errorf("malformed frame (id=%q type=%q)", ev.ID, ev.Type)
		}
		return ev, nil
	}
}

// Ping sends an application-layer keepalive. The supervisor calls it on
// a 30s tick; if no pong arrives before the next tick, Read will return
// an error (WS lower-level read deadline kicks in) and the supervisor
// reconnects.
func (c *StreamClient) Ping(ctx context.Context) error {
	return c.writeJSON(ctx, PingFrame{Type: "ping"})
}

// Close sends a clean 1001 ("going away") so the counterparty marks the
// subscription as gracefully disconnected. Best-effort; if the
// connection is already dead this is a no-op error which we ignore.
func (c *StreamClient) Close() {
	_ = c.conn.Close(websocket.StatusGoingAway, "client shutdown")
}

func (c *StreamClient) writeJSON(ctx context.Context, v any) error {
	buf, err := json.Marshal(v)
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return c.conn.Write(writeCtx, websocket.MessageText, buf)
}

func (c *StreamClient) readJSON(ctx context.Context, out any) error {
	buf, err := c.readBytes(ctx)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, out)
}

func (c *StreamClient) readBytes(ctx context.Context) ([]byte, error) {
	_, buf, err := c.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// DeriveStreamEndpoint maps a REST endpoint (http(s)://) to a WS
// endpoint (ws(s)://) on /v1/stream when the operator did not supply
// an explicit StreamEndpoint in config. Returns the input untouched if
// it already starts with ws:// or wss://.
func DeriveStreamEndpoint(restEndpoint string) (string, error) {
	u, err := url.Parse(restEndpoint)
	if err != nil {
		return "", fmt.Errorf("parse endpoint %q: %w", restEndpoint, err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
		// already a WS URL
	default:
		return "", fmt.Errorf("unsupported endpoint scheme %q", u.Scheme)
	}
	// Only append /v1/stream when the operator passed the REST base —
	// a complete `wss://host/v1/stream` URL must round-trip unchanged.
	if !strings.HasSuffix(u.Path, "/v1/stream") {
		u.Path = strings.TrimSuffix(u.Path, "/") + "/v1/stream"
	}
	return u.String(), nil
}
