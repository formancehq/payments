package client

import (
	"errors"
	"fmt"
	"strings"
)

// ErrPayableNotFound is returned when a payable lookup returns 404. The
// poll workflow uses errors.Is to keep polling instead of failing.
var ErrPayableNotFound = errors.New("payable not found")

// Error renders the Routable error response into a single line that keeps
// every field the API gave us, so logs surface enough context to debug
// 4xx/5xx without a network capture.
func (e ErrorResponse) Error() string {
	if e.Message == "" && e.Code == "" && len(e.Errors) == 0 {
		return "routable api error: empty body"
	}

	var b strings.Builder
	b.WriteString("routable api error")
	if e.Code != "" {
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
			if fe.Field != "" {
				fmt.Fprintf(&b, "%s: ", fe.Field)
			}
			b.WriteString(fe.Message)
		}
	}
	return b.String()
}
