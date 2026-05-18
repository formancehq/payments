package universal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
)

func TestResolveSubscribeList(t *testing.T) {
	t.Parallel()
	supported := map[string]struct{}{
		"payment.updated":   {},
		"payment.created":   {},
		"account.updated":   {},
		"balance.updated":   {},
	}

	t.Run("wildcard expands to every supported event in sorted order", func(t *testing.T) {
		t.Parallel()
		got, err := resolveSubscribeList([]string{"*"}, supported)
		if err != nil {
			t.Fatalf("unexpected: %v", err)
		}
		want := []string{"account.updated", "balance.updated", "payment.created", "payment.updated"}
		if !equalStrings(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
	})

	t.Run("explicit list intersects with supported", func(t *testing.T) {
		t.Parallel()
		got, err := resolveSubscribeList([]string{"payment.updated", "garbage", "account.updated"}, supported)
		// Unknown entries surface a non-fatal droppedEventsNotice so
		// the supervisor logs them; the resolved list is still valid.
		var notice *droppedEventsNotice
		if err != nil && !errors.As(err, &notice) {
			t.Fatalf("unexpected non-notice error: %v", err)
		}
		want := []string{"account.updated", "payment.updated"}
		if !equalStrings(got, want) {
			t.Fatalf("got %v want %v", got, want)
		}
		if notice == nil || len(notice.events) != 1 || notice.events[0] != "garbage" {
			t.Fatalf("expected dropped=[garbage], got %+v", notice)
		}
	})

	t.Run("empty offered list rejected", func(t *testing.T) {
		t.Parallel()
		if _, err := resolveSubscribeList(nil, supported); err == nil {
			t.Fatal("expected error for empty offered list")
		}
	})

	t.Run("offered with zero overlap rejected", func(t *testing.T) {
		t.Parallel()
		if _, err := resolveSubscribeList([]string{"nope"}, supported); err == nil {
			t.Fatal("expected error for empty intersection")
		}
	})
}

func TestBackoffBounded(t *testing.T) {
	t.Parallel()
	for attempt := 1; attempt <= 20; attempt++ {
		d := backoff(attempt)
		if d < 0 || d > streamReconnectMax {
			t.Fatalf("attempt %d: backoff %s out of bounds [0, %s]", attempt, d, streamReconnectMax)
		}
	}
}

func TestBackoffZeroAttemptClampsToOne(t *testing.T) {
	t.Parallel()
	d := backoff(0)
	if d > streamReconnectMin {
		t.Fatalf("attempt=0 must clamp to attempt=1 ceiling (%s), got %s", streamReconnectMin, d)
	}
}

// recordingDispatcher captures every loopback Dispatch call so the
// supervisor integration test can assert without standing up a real
// HTTP gateway.
type recordingDispatcher struct {
	mu    sync.Mutex
	calls []dispatchCall
}

type dispatchCall struct {
	URL       string
	Signature string
	Timestamp string
	Body      []byte
}

func (r *recordingDispatcher) Dispatch(_ context.Context, urlAbs, signature, timestamp string, body []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, dispatchCall{URL: urlAbs, Signature: signature, Timestamp: timestamp, Body: append([]byte(nil), body...)})
	return nil
}

func (r *recordingDispatcher) Calls() []dispatchCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]dispatchCall(nil), r.calls...)
}

// TestStreamSupervisorIntegration spins up a minimal in-test WS server,
// runs the supervisor against it, pushes one event, and asserts the
// loopback dispatcher saw exactly the expected signed POST.
func TestStreamSupervisorIntegration(t *testing.T) {
	t.Parallel()
	const (
		apiKey = "test-key"
		secret = "test-secret"
	)

	var dialed atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dialed.Add(1)
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{client.StreamSubProtocol}})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "test done")

		ctx := r.Context()
		_, raw, err := c.Read(ctx)
		if err != nil {
			return
		}
		var hello client.HelloFrame
		if err := json.Unmarshal(raw, &hello); err != nil {
			t.Errorf("hello decode: %v", err)
			return
		}
		ackBuf, _ := json.Marshal(client.HelloAck{Type: "hello-ack", Accepted: hello.Events, ServerTime: time.Now().UTC()})
		_ = c.Write(ctx, websocket.MessageText, ackBuf)

		evBuf, _ := json.Marshal(client.WebhookEvent{ID: "evt-int-1", Type: "payment.updated", CreatedAt: time.Now().UTC()})
		_ = c.Write(ctx, websocket.MessageText, evBuf)

		// Hold the connection open so the supervisor's read loop has time
		// to deliver the frame to the dispatcher before close.
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	rec := &recordingDispatcher{}
	logger := logging.NewDefaultLogger(streamTestWriter{t}, true, false, false)
	sup, err := newStreamSupervisor(
		logger,
		"universal-test",
		Config{Endpoint: srv.URL, APIKey: apiKey, WebhookSharedSecret: secret, StreamEndpoint: wsURL},
		client.Features{EventStream: "wss", StreamEvents: []string{"*"}},
		"http://gateway.test",
		supportedWebhooks,
		rec,
	)
	if err != nil {
		t.Fatalf("newStreamSupervisor: %v", err)
	}

	sup.Start(context.Background())
	defer sup.Stop()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(rec.Calls()) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	calls := rec.Calls()
	if len(calls) == 0 {
		t.Fatal("dispatcher saw no calls after 3s")
	}
	call := calls[0]
	if !strings.HasSuffix(call.URL, "/payment/updated") {
		t.Fatalf("loopback URL %q must end with the supportedWebhook URLPath", call.URL)
	}
	if call.Signature == "" || call.Timestamp == "" {
		t.Fatalf("loopback signature/timestamp must be set: %+v", call)
	}
	var ev struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(call.Body, &ev); err != nil {
		t.Fatalf("body decode: %v", err)
	}
	if ev.ID != "evt-int-1" || ev.Type != "payment.updated" {
		t.Fatalf("body mismatch: %+v", ev)
	}
	if dialed.Load() == 0 {
		t.Fatal("expected at least one dial against the WS test server")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// streamTestWriter funnels log output through *testing.T so failures
// surface inline with the test that produced them. Kept tiny —
// duplicating testWriter in config_test.go is cheaper than exporting it
// since we're in the white-box (universal) package and that one is
// black-box (universal_test).
type streamTestWriter struct{ t *testing.T }

func (w streamTestWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s", p)
	return len(p), nil
}
