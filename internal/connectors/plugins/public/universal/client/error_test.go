package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/plugins"
)

// Both envelope shapes from the connector skill's Phase 1.5 traps are
// accepted by the same Error struct via tag-based unmarshalling.

func TestErrorAcceptsLegacyEnvelope(t *testing.T) {
	t.Parallel()
	body := `{"message":"bad input","errors":[{"field":"reference","message":"required"}]}`
	var e Error
	if err := json.Unmarshal([]byte(body), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	e.HTTPStatus = http.StatusBadRequest
	msg := e.Error()
	if !strings.Contains(msg, "bad input") || !strings.Contains(msg, "reference") {
		t.Fatalf("missing context: %q", msg)
	}
}

func TestErrorAcceptsRFC7807Envelope(t *testing.T) {
	t.Parallel()
	body := `{"type":"about:blank","title":"Validation","status":422,"detail":"amount is negative","errors":[{"path":"amount","detail":"must be positive"}]}`
	var e Error
	if err := json.Unmarshal([]byte(body), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	e.HTTPStatus = 422
	msg := e.Error()
	if !strings.Contains(msg, "Validation") || !strings.Contains(msg, "amount") {
		t.Fatalf("missing context: %q", msg)
	}
}

func TestErrorIsMappingToSentinels(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		status int
		want   error
	}{
		{"429 → ratelimit", http.StatusTooManyRequests, plugins.ErrUpstreamRatelimit},
		{"408 → timeout", http.StatusRequestTimeout, plugins.ErrUpstreamTimeout},
		{"504 → timeout", http.StatusGatewayTimeout, plugins.ErrUpstreamTimeout},
		{"400 → invalid client request", http.StatusBadRequest, plugins.ErrInvalidClientRequest},
		{"500 → server error", http.StatusInternalServerError, httpwrapper.ErrStatusCodeServerError},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := &Error{HTTPStatus: c.status}
			if !errors.Is(e, c.want) {
				t.Fatalf("status %d expected to satisfy %v", c.status, c.want)
			}
		})
	}
}

func TestErrorUnwrapPropagates(t *testing.T) {
	t.Parallel()
	inner := errors.New("transport boom")
	e := &Error{HTTPStatus: http.StatusBadRequest, Underlying: inner}
	if !errors.Is(e, inner) {
		t.Fatal("Unwrap not exposing underlying error")
	}
}
