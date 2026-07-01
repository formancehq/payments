package client

import (
	"errors"
	"testing"
)

func TestAPIErrorMessage(t *testing.T) {
	t.Parallel()
	e := &APIError{Endpoint: "/0/private/Balance", Code: "EAPI:Invalid key", All: []string{"EAPI:Invalid key"}}
	if !IsAPIError(e) {
		t.Error("IsAPIError must match")
	}
	wrapped := errors.New("wrapped: " + e.Error())
	if IsAPIError(wrapped) {
		t.Error("plain error must not match IsAPIError")
	}
}

func TestIsFatalAuthError(t *testing.T) {
	t.Parallel()
	for _, code := range []string{
		"EAPI:Invalid key", "EAPI:Invalid signature", "EAPI:Bad request", "EGeneral:Permission denied",
	} {
		err := &APIError{Endpoint: "/x", Code: code, All: []string{code}}
		if !IsFatalAuthError(err) {
			t.Errorf("%q should be fatal", code)
		}
	}
	// Invalid nonce is a transient cross-pod race, not a config error.
	for _, code := range []string{"EService:Unavailable", "EOrder:Insufficient funds", "EQuery:Unknown asset pair", "EAPI:Invalid nonce"} {
		err := &APIError{Endpoint: "/x", Code: code, All: []string{code}}
		if IsFatalAuthError(err) {
			t.Errorf("%q should not be fatal", code)
		}
	}
}

func TestIsRetriableError(t *testing.T) {
	t.Parallel()
	for _, code := range []string{"EAPI:Invalid nonce", "EAPI:Rate limit exceeded", "EService:Throttled: 1700000000"} {
		if !IsRetriableError(&APIError{Code: code, All: []string{code}}) {
			t.Errorf("%q should be retriable", code)
		}
	}
	for _, code := range []string{"EAPI:Invalid key", "EQuery:Unknown asset pair"} {
		if IsRetriableError(&APIError{Code: code, All: []string{code}}) {
			t.Errorf("%q should not be retriable", code)
		}
	}
}

func TestErrorResponseMessage(t *testing.T) {
	t.Parallel()
	r := ErrorResponse{Errors: []string{"EAPI:Invalid key", "EAPI:Other"}}
	if r.Message() != "EAPI:Invalid key" {
		t.Errorf("Message=%q", r.Message())
	}
	if (ErrorResponse{}).Message() != "" {
		t.Error("empty error must yield empty message")
	}
}
