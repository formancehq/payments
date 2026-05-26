package client

import "errors"

// ErrorResponse accepts both Bitstamp envelope shapes that co-exist on
// the v2 API:
//
//   - legacy: {"status":"error","reason":"...","code":"API..."}
//   - newer:  {"code":"API...","message":"..."}
//
// Endpoints may serve either form depending on which microservice
// answers, so the connector tolerates both silently.
type ErrorResponse struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
	Code   string `json:"code"`
	Msg    string `json:"message"`
}

// Message returns the most specific human-readable description
// available on the envelope.
func (e ErrorResponse) Message() string {
	switch {
	case e.Msg != "":
		return e.Msg
	case e.Reason != "":
		return e.Reason
	default:
		return e.Code
	}
}

// ErrCodeDerivativesUnsupported is the documented Bitstamp error code
// returned when a derivatives-gated endpoint is invoked by a spot-only
// account (or vice-versa).
const ErrCodeDerivativesUnsupported = "API5506"

// DerivativesUnsupportedError is returned by client methods when a
// request hits a derivatives-gated path on a spot-only account. The
// connector is spot-only by design; callers should treat this as an
// empty result rather than propagating it as a failure.
//
// None of the currently-implemented client methods (account_balances,
// user_transactions, currencies, open_orders, order_status) are
// themselves derivatives-gated, so this is preventive plumbing kept
// here so a future contributor extending the surface (e.g. adding
// /api/v2/trade_history/) gets the right behaviour by default.
type DerivativesUnsupportedError struct {
	Endpoint string
	Message  string
}

func (e *DerivativesUnsupportedError) Error() string {
	return "bitstamp: derivatives not supported on " + e.Endpoint + ": " + e.Message
}

// IsDerivativesUnsupportedError reports whether err is (or wraps) a
// DerivativesUnsupportedError.
func IsDerivativesUnsupportedError(err error) bool {
	var d *DerivativesUnsupportedError
	return errors.As(err, &d)
}

// NotFoundError is returned by client methods when the server responds
// with HTTP 404. For account_order_data this typically means the market
// is not available for the authenticated account.
type NotFoundError struct {
	Endpoint string
	Message  string
}

func (e *NotFoundError) Error() string {
	return "bitstamp: not found on " + e.Endpoint + ": " + e.Message
}

// IsNotFoundError reports whether err is (or wraps) a NotFoundError.
func IsNotFoundError(err error) bool {
	var n *NotFoundError
	return errors.As(err, &n)
}
