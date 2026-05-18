package client

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/plugins"
)

// Error is the universal-contract error envelope. It accepts both shapes:
// RFC 7807 (`application/problem+json`) and the legacy `{message, errors}`
// shape — same struct, tags overlap, no custom UnmarshalJSON needed.
// HTTPStatus and Underlying are stamped by client.do after the fact.
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
	return strings.Join(filterEmpty(parts), " - ")
}

func (e *Error) Unwrap() error { return e.Underlying }

// Is collapses the wire status onto plugin sentinels so engine activities
// classify retryable vs terminal failures consistently. Mirrors the
// registry's translateError + httpwrapper.defaultHttpErrorCheckerFn so an
// upstream change shows up in one place. 408/421/423/425 are explicitly
// NOT generic-4xx — they map to ErrUpstreamTimeout/ErrUpstreamRetryAfter
// so Temporal retries them instead of marking the activity terminal.
func (e *Error) Is(target error) bool {
	switch {
	case errors.Is(target, plugins.ErrUpstreamRatelimit):
		return e.HTTPStatus == http.StatusTooManyRequests
	case errors.Is(target, plugins.ErrUpstreamTimeout):
		switch e.HTTPStatus {
		case http.StatusRequestTimeout, http.StatusMisdirectedRequest, http.StatusGatewayTimeout:
			return true
		}
		return false
	case errors.Is(target, plugins.ErrUpstreamRetryAfter):
		return e.HTTPStatus == http.StatusLocked || e.HTTPStatus == http.StatusTooEarly
	case errors.Is(target, plugins.ErrInvalidClientRequest):
		if e.HTTPStatus < http.StatusBadRequest || e.HTTPStatus >= http.StatusInternalServerError {
			return false
		}
		switch e.HTTPStatus {
		case http.StatusRequestTimeout, http.StatusMisdirectedRequest, http.StatusLocked, http.StatusTooEarly, http.StatusTooManyRequests:
			return false
		}
		return true
	case errors.Is(target, httpwrapper.ErrStatusCodeServerError):
		return e.HTTPStatus >= http.StatusInternalServerError
	}
	return false
}

func filterEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
