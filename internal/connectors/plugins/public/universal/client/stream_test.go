package client

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestSignHelloDeterministic(t *testing.T) {
	t.Parallel()
	s1 := SignHello("s", "2026-05-18T12:34:56Z", "n", []string{"a", "b"})
	s2 := SignHello("s", "2026-05-18T12:34:56Z", "n", []string{"a", "b"})
	if s1 != s2 {
		t.Fatalf("SignHello must be deterministic; got %s vs %s", s1, s2)
	}
	// Sanity: hex-decodable, 32 bytes (HMAC-SHA256).
	b, err := hex.DecodeString(s1)
	if err != nil {
		t.Fatalf("signature not hex: %v", err)
	}
	if len(b) != 32 {
		t.Fatalf("HMAC-SHA256 must be 32 bytes, got %d", len(b))
	}
}

func TestSignHelloDifferentNoncesProduceDifferentSignatures(t *testing.T) {
	t.Parallel()
	s1 := SignHello("s", "2026-05-18T12:34:56Z", "n1", []string{"a"})
	s2 := SignHello("s", "2026-05-18T12:34:56Z", "n2", []string{"a"})
	if s1 == s2 {
		t.Fatal("different nonces must produce different signatures")
	}
}

func TestNewNonceUniqueness(t *testing.T) {
	t.Parallel()
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		n, err := NewNonce()
		if err != nil {
			t.Fatalf("NewNonce: %v", err)
		}
		if len(n) != 32 {
			t.Fatalf("nonce must be 32 hex chars (16 bytes), got %d", len(n))
		}
		if _, dup := seen[n]; dup {
			t.Fatalf("duplicate nonce in 1000 draws (vanishingly unlikely): %s", n)
		}
		seen[n] = struct{}{}
	}
}

func TestDeriveStreamEndpoint(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
		err      bool
	}{
		{"http://localhost:8080", "ws://localhost:8080/v1/stream", false},
		{"https://api.example.com", "wss://api.example.com/v1/stream", false},
		{"https://api.example.com/", "wss://api.example.com/v1/stream", false},
		{"https://api.example.com/base/", "wss://api.example.com/base/v1/stream", false},
		{"ws://already.ws/v1/stream", "ws://already.ws/v1/stream", false},
		{"wss://already.wss/v1/stream", "wss://already.wss/v1/stream", false},
		{"ftp://nope.example", "", true},
	}
	for _, c := range cases {
		c := c
		t.Run(c.in, func(t *testing.T) {
			t.Parallel()
			got, err := DeriveStreamEndpoint(c.in)
			if c.err {
				if err == nil {
					t.Fatalf("expected error for %q, got %q", c.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

// TestDialStreamHappyPath spins up a minimal WS server that accepts the
// signed hello and emits one WebhookEvent. Asserts the client parses
// the ack, reads the event, and Close drives a clean WS shutdown.
func TestDialStreamHappyPath(t *testing.T) {
	t.Parallel()
	const (
		apiKey = "test-key"
		secret = "test-secret"
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{StreamSubProtocol}})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "test done")
		ctx := r.Context()

		// Read hello.
		_, raw, err := c.Read(ctx)
		if err != nil {
			t.Errorf("read hello: %v", err)
			return
		}
		var hello HelloFrame
		if err := json.Unmarshal(raw, &hello); err != nil {
			t.Errorf("decode hello: %v", err)
			return
		}
		expectedSig := SignHello(secret, hello.Timestamp, hello.Nonce, hello.Events)
		if hello.Signature != expectedSig {
			t.Errorf("signature mismatch on server side: got %q want %q", hello.Signature, expectedSig)
			return
		}

		// Ack.
		ack := HelloAck{Type: "hello-ack", Accepted: hello.Events, ServerTime: time.Now().UTC()}
		ackBuf, _ := json.Marshal(ack)
		if err := c.Write(ctx, websocket.MessageText, ackBuf); err != nil {
			t.Errorf("ack write: %v", err)
			return
		}

		// Push one event.
		ev := WebhookEvent{ID: "evt-1", Type: "payment.updated", CreatedAt: time.Now().UTC()}
		evBuf, _ := json.Marshal(ev)
		_ = c.Write(ctx, websocket.MessageText, evBuf)
	}))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, ack, err := DialStream(ctx, StreamDialConfig{
		Endpoint: wsURL,
		APIKey:   apiKey,
		Secret:   secret,
		Events:   []string{"payment.updated"},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
	if ack == nil || ack.Type != "hello-ack" {
		t.Fatalf("missing/wrong ack: %+v", ack)
	}
	ev, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("read event: %v", err)
	}
	if ev.ID != "evt-1" || ev.Type != "payment.updated" {
		t.Fatalf("event roundtrip wrong: %+v", ev)
	}
}

func TestDialStreamRequiresSecret(t *testing.T) {
	t.Parallel()
	_, _, err := DialStream(context.Background(), StreamDialConfig{Endpoint: "ws://x", APIKey: "k", Events: []string{"a"}})
	if err == nil {
		t.Fatal("expected error for missing secret")
	}
}

func TestDialStreamRequiresEvents(t *testing.T) {
	t.Parallel()
	_, _, err := DialStream(context.Background(), StreamDialConfig{Endpoint: "ws://x", APIKey: "k", Secret: "s"})
	if err == nil {
		t.Fatal("expected error for empty events list")
	}
}
