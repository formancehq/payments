package client

import (
	"errors"
	"fmt"
	"strings"
)

// ErrPayableNotFound is returned when a payable lookup returns 404. The
// poll workflow uses errors.Is to keep polling instead of failing.
var ErrPayableNotFound = errors.New("payable not found")

// hasContent reports whether the envelope carries anything worth surfacing
// in a wrapped error message. An "empty" envelope is the zero value, and
// appending it to a transport error just produces a misleading suffix.
func (e ErrorResponse) hasContent() bool {
	if e.Title != "" || e.Message != "" || e.Code != "" || e.RequestID != "" {
		return true
	}
	for _, fe := range e.Errors {
		if fe.Detail != "" || fe.Message != "" || fe.Path != "" || fe.Field != "" {
			return true
		}
	}
	return false
}

// Error renders the Routable error response into a single line that keeps
// every field the API gave us, so logs surface enough context to debug
// 4xx/5xx without a network capture. Handles both the legacy
// {object:"Error", message, errors[].field} envelope and the v1 RFC 7807
// {title, status, errors[].path/detail} application/problem+json envelope.
func (e ErrorResponse) Error() string {
	if !e.hasContent() {
		return "routable api error: empty body"
	}

	var b strings.Builder
	b.WriteString("routable api error")

	// Header: prefer v1's title; fall back to the legacy code/message.
	switch {
	case e.Title != "":
		fmt.Fprintf(&b, " [%s", e.Title)
		if e.Status != 0 {
			fmt.Fprintf(&b, " %d", e.Status)
		}
		b.WriteString("]")
	case e.Code != "":
		fmt.Fprintf(&b, " [%s]", e.Code)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}

	if len(e.Errors) > 0 {
		b.WriteString("; details:")
		for i, fe := range e.Errors {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(" ")
			b.WriteString(fe.format())
		}
	}

	if e.RequestID != "" {
		fmt.Fprintf(&b, " (request_id=%s)", e.RequestID)
	}
	return b.String()
}

// format renders one field-level error, accepting both the legacy
// {field, message} shape and the v1 {path, detail} shape.
func (fe FieldError) format() string {
	loc := fe.Path
	if loc == "" {
		loc = fe.Field
	}
	msg := fe.Detail
	if msg == "" {
		msg = fe.Message
	}
	switch {
	case loc != "" && msg != "":
		return loc + ": " + msg
	case loc != "":
		return loc
	default:
		return msg
	}
}
