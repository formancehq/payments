package client

import (
	"bytes"

	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/form"
)

//go:generate mockgen -source backend.go -destination backend_generated.go -package client . Backend
type Backend interface {
	Call(method, path, key string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error
	CallStreaming(method, path, key string, params stripe.ParamsContainer, v stripe.StreamingLastResponseSetter) error
	CallRaw(string, string, string, *form.Values, *stripe.Params, stripe.LastResponseSetter) error
	CallMultipart(method, path, key, boundary string, body *bytes.Buffer, params *stripe.Params, v stripe.LastResponseSetter) error
	SetMaxNetworkRetries(maxNetworkRetries int64)
}
