package client

import (
	"errors"
	"fmt"
)

var (
	ErrWebhookUrlMissing              = errors.New("webhook url is not set")
	ErrWebhookHeaderXSignatureMissing = errors.New("missing X-Signature-Sha256 header")
	ErrWebhookNameUnknown             = errors.New("unknown webhook name")
	ErrWebhookRequestFailed           = errors.New("failed to create webhooks request")
)

type columnError struct {
	Code             string `json:"code"`
	DocumentationUrl string `json:"documentation_url"`
	Message          string `json:"message"`
	Type             string `json:"type"`
}

func (ce *columnError) Error() error {
	return fmt.Errorf("%s: %s", ce.Code, ce.Message)
}
