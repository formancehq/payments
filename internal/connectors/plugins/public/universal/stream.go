package universal

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
)

// Stream-supervisor tuning. Values picked to match the rest of the
// connector (signatureTolerance for clock skew, PAGE_SIZE-ish buffer
// depth so a slow gateway doesn't OOM the worker).
const (
	streamBufferDepth    = 64
	streamPingInterval   = 30 * time.Second
	streamReconnectMin   = 1 * time.Second
	streamReconnectMax   = 60 * time.Second
	streamLoopbackPOSTTo = 15 * time.Second
)

// streamEventWildcard is the sentinel the counterparty sends in
// features.streamEvents to mean "everything I publish on webhooks I
// also publish on the stream". Resolution expands it to the full
// supportedWebhooks slice.
const streamEventWildcard = "*"

// streamSupervisor owns the WebSocket connection lifecycle for one
// Plugin instance. Started by CreateWebhooks when the counterparty
// advertises features.eventStream=="wss"; stopped by Uninstall.
//
// One goroutine handles the dial / read / reconnect loop. Each
// inbound frame is funnelled into a bounded channel; a second
// goroutine drains the channel and HMAC-signs + POSTs the frame back
// to the engine's own webhook endpoint, reusing the exact same
// VerifyWebhook/TranslateWebhook pipeline as HTTP-pushed deliveries.
//
// At-least-once: the engine's WebhookIdempotencyKey dedup handles
// duplicate deliveries (across pods or after reconnect-without-cursor).
type streamSupervisor struct {
	logger     logging.Logger
	connector  string
	dial       client.StreamDialConfig
	gatewayURL string
	secret     string
	urlPaths   map[string]string // event name -> URL suffix
	dispatcher streamDispatcher

	startOnce sync.Once
	cancel    context.CancelFunc
	done      chan struct{}
}

// streamDispatcher abstracts the loopback POST so tests can substitute
// an in-process counter. Production wires it to http.DefaultClient.
type streamDispatcher interface {
	Dispatch(ctx context.Context, urlAbs, signature, timestamp string, body []byte) error
}

// resolveSubscribeList intersects the counterparty's declared stream
// events with the engine-routable event set. ["*"] sentinel expands to
// every supportedWebhookNames entry in deterministic order. Errors are
// plain — the caller (Plugin.maybeStartStream) wraps them with
// models.ErrInvalidConfig so the engine surfaces install-time
// misconfiguration as non-retryable.
//
// Unknown event names in `offered` are dropped (not an error): the
// engine can simply not route them. If the intersection is empty the
// install MUST fail — silent no-subscription is the misconfiguration
// we are guarding against.
func resolveSubscribeList(offered []string, supported map[string]struct{}) ([]string, error) {
	if len(offered) == 0 {
		return nil, errors.New("features.streamEvents must be non-empty when features.eventStream=\"wss\"")
	}
	if len(offered) == 1 && offered[0] == streamEventWildcard {
		out := make([]string, 0, len(supported))
		for name := range supported {
			out = append(out, name)
		}
		sort.Strings(out)
		return out, nil
	}
	out := make([]string, 0, len(offered))
	dropped := make([]string, 0)
	for _, name := range offered {
		if _, ok := supported[name]; !ok {
			dropped = append(dropped, name)
			continue
		}
		out = append(out, name)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("features.streamEvents %v has no overlap with engine-routable events", offered)
	}
	sort.Strings(out)
	if len(dropped) > 0 {
		// Surfaced via the supervisor's logger on construction so the
		// operator sees what was dropped without failing the install
		// when at least one event remains usable.
		sort.Strings(dropped)
		return out, &droppedEventsNotice{events: dropped}
	}
	return out, nil
}

// droppedEventsNotice is a non-fatal warning surfaced from
// resolveSubscribeList alongside the resolved list. The supervisor
// constructor logs it at warn level and discards it; nothing
// upstream treats it as an error.
type droppedEventsNotice struct{ events []string }

func (n *droppedEventsNotice) Error() string {
	return "stream events dropped (unknown to engine): " + strings.Join(n.events, ",")
}

// newStreamSupervisor builds a supervisor; call Start to spawn the
// goroutines. Returns an error if the dial endpoint can't be derived
// or the subscribe list is unusable — both surfaced as
// models.ErrInvalidConfig at install time.
func newStreamSupervisor(
	logger logging.Logger,
	connector string,
	cfg Config,
	features client.Features,
	gatewayBaseURL string,
	supported []supportedWebhook,
	dispatcher streamDispatcher,
) (*streamSupervisor, error) {
	endpoint := cfg.StreamEndpoint
	if endpoint == "" {
		derived, err := client.DeriveStreamEndpoint(cfg.Endpoint)
		if err != nil {
			return nil, err
		}
		endpoint = derived
	}

	supportedSet := make(map[string]struct{}, len(supported))
	urlPaths := make(map[string]string, len(supported))
	for _, w := range supported {
		supportedSet[w.Name] = struct{}{}
		urlPaths[w.Name] = w.URLPath
	}

	events, err := resolveSubscribeList(features.StreamEvents, supportedSet)
	supLogger := logger.WithField("connector", connector).WithField("component", "stream")
	if events == nil && err != nil {
		return nil, err
	}
	if dn := (*droppedEventsNotice)(nil); errors.As(err, &dn) {
		supLogger.WithField("dropped_events", dn.events).Errorf("stream events declared by counterparty are unknown to the engine — they will not be subscribed to")
	}

	if _, err := url.Parse(gatewayBaseURL); err != nil || gatewayBaseURL == "" {
		return nil, fmt.Errorf("invalid gateway base URL %q for loopback: %w", gatewayBaseURL, err)
	}

	return &streamSupervisor{
		logger:    supLogger,
		connector: connector,
		dial: client.StreamDialConfig{
			Endpoint: endpoint,
			APIKey:   cfg.APIKey,
			Secret:   cfg.WebhookSharedSecret,
			Events:   events,
		},
		gatewayURL: gatewayBaseURL,
		secret:     cfg.WebhookSharedSecret,
		urlPaths:   urlPaths,
		dispatcher: dispatcher,
	}, nil
}

// Start spawns the reader and dispatcher goroutines. Idempotent: a
// repeated Start (e.g. race between CreateWebhooks and the lazy
// FetchNext* recovery path on a pod restart) is a silent no-op.
// Callers that want strict "started-once-per-supervisor" semantics
// should rely on the lifecycle owner — Plugin — to gate construction.
func (s *streamSupervisor) Start(parent context.Context) {
	s.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(parent)
		s.cancel = cancel
		s.done = make(chan struct{})

		frames := make(chan client.WebhookEvent, streamBufferDepth)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			s.readerLoop(ctx, frames)
		}()
		go func() {
			defer wg.Done()
			s.dispatchLoop(ctx, frames)
		}()
		go func() {
			wg.Wait()
			close(s.done)
		}()
		s.logger.WithFields(map[string]any{
			"endpoint": s.dial.Endpoint,
			"events":   s.dial.Events,
			"gateway":  s.gatewayURL,
		}).Info("stream supervisor started")
	})
}

// Stop signals the supervisor to drain and disconnect. Blocks until
// both goroutines exit. Safe to call even when Start never ran (no-op)
// or twice (second call is a no-op). Never panics.
func (s *streamSupervisor) Stop() {
	if s.cancel == nil || s.done == nil {
		return
	}
	s.cancel()
	<-s.done
	s.logger.Info("stream supervisor stopped")
}

// readerLoop owns the dial → read-until-error → reconnect cycle. Frames
// are pushed onto the channel; ctx cancellation drains the loop on
// shutdown.
func (s *streamSupervisor) readerLoop(ctx context.Context, frames chan<- client.WebhookEvent) {
	attempt := 0
	for {
		if ctx.Err() != nil {
			return
		}
		conn, ack, err := client.DialStream(ctx, s.dial)
		if err != nil {
			attempt++
			delay := backoff(attempt)
			s.logger.WithFields(map[string]any{
				"attempt":  attempt,
				"delay_ms": delay.Milliseconds(),
				"error":    err.Error(),
			}).Errorf("stream dial failed")
			if !sleep(ctx, delay) {
				return
			}
			continue
		}
		s.logger.WithFields(map[string]any{
			"accepted":    ack.Accepted,
			"server_time": ack.ServerTime.Format(time.RFC3339),
			"attempt":     attempt + 1,
		}).Info("stream connected")
		attempt = 0

		s.runConnection(ctx, conn, frames)
		// runConnection returned -> connection closed; either ctx is
		// done (clean shutdown) or upstream dropped (reconnect).
	}
}

// runConnection blocks until the WS errors or ctx cancels. Drives a
// 30s ping ticker on a side goroutine. On exit, sends a clean close.
func (s *streamSupervisor) runConnection(ctx context.Context, conn *client.StreamClient, frames chan<- client.WebhookEvent) {
	pingCtx, cancelPing := context.WithCancel(ctx)
	defer cancelPing()
	defer conn.Close()

	go func() {
		t := time.NewTicker(streamPingInterval)
		defer t.Stop()
		for {
			select {
			case <-pingCtx.Done():
				return
			case <-t.C:
				if err := conn.Ping(pingCtx); err != nil {
					s.logger.Debugf("stream ping failed: %s", err)
					return
				}
			}
		}
	}()

	for {
		ev, err := conn.Read(ctx)
		if err != nil {
			if ctx.Err() == nil {
				s.logger.WithField("error", err).Error("stream read error, will reconnect")
			}
			return
		}
		select {
		case frames <- ev:
		case <-ctx.Done():
			return
		default:
			// Buffer full → drop, but log loudly. The engine's
			// FetchNext* polling backfills on the next cycle, so a
			// dropped notification doesn't drop the resource.
			s.logger.WithFields(map[string]any{
				"event": ev.Type,
				"id":    ev.ID,
			}).Errorf("stream dispatch buffer full (depth=%d), dropping event", streamBufferDepth)
		}
	}
}

// dispatchLoop drains the frame channel, signs each frame, and POSTs
// it to the connector's own webhook URL — reusing the existing
// VerifyWebhook+TranslateWebhook+RunHandleWebhooks pipeline. On
// dispatch error we log and continue; the engine's idempotency dedup
// makes retries safe but the supervisor doesn't retry itself (the
// counterparty will likely re-emit on reconnect).
func (s *streamSupervisor) dispatchLoop(ctx context.Context, frames <-chan client.WebhookEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-frames:
			if !ok {
				return
			}
			s.dispatchOne(ctx, ev)
		}
	}
}

func (s *streamSupervisor) dispatchOne(ctx context.Context, ev client.WebhookEvent) {
	urlPath, ok := s.urlPaths[ev.Type]
	if !ok {
		s.logger.WithField("event", ev.Type).Debug("stream event without engine route, dropping")
		return
	}
	abs, err := url.JoinPath(s.gatewayURL, urlPath)
	if err != nil {
		s.logger.WithFields(map[string]any{"event": ev.Type, "error": err.Error()}).Error("stream loopback URL join failed")
		return
	}
	body, err := json.Marshal(ev)
	if err != nil {
		s.logger.WithFields(map[string]any{"event": ev.Type, "error": err}).Error("stream loopback marshal failed")
		return
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	signature := signWebhookBody(s.secret, timestamp, body)

	dispatchCtx, cancel := context.WithTimeout(ctx, streamLoopbackPOSTTo)
	defer cancel()
	if err := s.dispatcher.Dispatch(dispatchCtx, abs, signature, timestamp, body); err != nil {
		s.logger.WithFields(map[string]any{
			"event": ev.Type,
			"id":    ev.ID,
			"error": err.Error(),
		}).Errorf("stream loopback dispatch failed")
		return
	}
	s.logger.WithFields(map[string]any{"event": ev.Type, "id": ev.ID}).Debug("stream event dispatched")
}

// signWebhookBody mirrors verifyHMACSHA256 in webhooks.go (the inverse
// path). One signing primitive — same shared secret, same hex output.
func signWebhookBody(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// httpDispatcher is the production streamDispatcher: POSTs the signed
// frame to the connector's own webhook endpoint through the local
// gateway, reusing every layer of the existing webhook pipeline.
type httpDispatcher struct {
	client *http.Client
}

func newHTTPDispatcher() *httpDispatcher {
	return &httpDispatcher{
		client: &http.Client{Timeout: streamLoopbackPOSTTo},
	}
}

func (d *httpDispatcher) Dispatch(ctx context.Context, absURL, signature, timestamp string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, absURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(WebhookHeaderSignature, signature)
	req.Header.Set(WebhookHeaderTimestamp, timestamp)
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("gateway returned %d", resp.StatusCode)
	}
	return nil
}

// backoff returns the sleep duration for reconnect attempt n (1-based)
// using full jitter: sleep ∈ [0, min(max, base * 2^(n-1))]. Matches the
// AWS-recommended algorithm and prevents thundering-herd on a rolling
// deploy of N pods.
func backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	ceil := streamReconnectMin << (attempt - 1)
	if ceil <= 0 || ceil > streamReconnectMax {
		ceil = streamReconnectMax
	}
	return time.Duration(rand.Int64N(int64(ceil)))
}

// sleep waits for d or ctx cancellation, returning false on cancel.
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// Compile-time guarantee that the production dispatcher satisfies the
// interface used by the supervisor.
var _ streamDispatcher = (*httpDispatcher)(nil)
