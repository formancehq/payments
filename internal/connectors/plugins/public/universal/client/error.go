package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/plugins"
)

// Error is the universal-contract error envelope. It accepts both shapes the
// Phase-1.5 traps section of the connector skill calls out:
//
//   1. RFC 7807 (application/problem+json):
//      { "type": "...", "title": "...", "status": 400, "detail": "...",
//        "instance": "...", "errors": [{ "path": "...", "detail": "..." }] }
//
//   2. Legacy:
//      { "message": "...", "errors": [{ "field": "...", "message": "..." }] }
//
// We unmarshal opportunistically: a request that returns either shape ends up
// in the same Go struct so callers don't need to care which envelope was
// served. The HTTPStatus and Underlying fields are stamped by client.do
// after the fact.
type Error struct {
	HTTPStatus int   `json:"-"`
	Underlying error `json:"-"`

	Type     string        `json:"type,omitempty"`
	Title    string        `json:"title,omitempty"`
	Status   int           `json:"status,omitempty"`
	Detail   string        `json:"detail,omitempty"`
	Instance string        `json:"instance,omitempty"`
	Message  string        `json:"message,omitempty"`
	Errors   []ErrorDetail `json:"errors,omitempty"`
}

type ErrorDetail struct {
	Path    string `json:"path,omitempty"`
	Field   string `json:"field,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Message string `json:"message,omitempty"`
}

func (e *Error) Error() string {
	parts := []string{fmt.Sprintf("universal client: HTTP %d", e.HTTPStatus)}
	if e.Title != "" {
		parts = append(parts, e.Title)
	}
	if e.Detail != "" {
		parts = append(parts, e.Detail)
	} else if e.Message != "" {
		parts = append(parts, e.Message)
	}
	for _, d := range e.Errors {
		field := d.Path
		if field == "" {
			field = d.Field
		}
		msg := d.Detail
		if msg == "" {
			msg = d.Message
		}
		if field != "" || msg != "" {
			parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
		}
	}
	if len(parts) == 1 && e.Underlying != nil {
		parts = append(parts, e.Underlying.Error())
	}
	return joinNonEmpty(parts, " - ")
}

func (e *Error) Unwrap() error { return e.Underlying }

// Is collapses the universal Error into the canonical plugin sentinels so
// engine activities can map the failure to retryable/non-retryable behaviour.
// We never expose raw upstream payloads through these — only the sentinel.
func (e *Error) Is(target error) bool {
	switch {
	case errors.Is(target, plugins.ErrUpstreamRatelimit):
		return e.HTTPStatus == http.StatusTooManyRequests
	case errors.Is(target, plugins.ErrUpstreamTimeout):
		return e.HTTPStatus == http.StatusRequestTimeout || e.HTTPStatus == http.StatusGatewayTimeout
	case errors.Is(target, plugins.ErrInvalidClientRequest):
		return e.HTTPStatus >= 400 && e.HTTPStatus < 500 && e.HTTPStatus != http.StatusTooManyRequests
	case errors.Is(target, httpwrapper.ErrStatusCodeServerError):
		return e.HTTPStatus >= 500
	}
	return false
}

func joinNonEmpty(parts []string, sep string) string {
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	switch len(out) {
	case 0:
		return ""
	case 1:
		return out[0]
	}
	res := out[0]
	for _, p := range out[1:] {
		res += sep + p
	}
	return res
}

// stub usage of json.Unmarshal so the file imports json if we later need
// custom unmarshalling (currently the standard tag-driven path covers both
// envelope shapes — a single struct with all fields set as omitempty).
var _ = json.Unmarshal
