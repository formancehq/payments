package client

import (
	"errors"
	"fmt"
)

var (
	ErrWebhookUrlMissing          = errors.New("webhook url is not set")
	ErrColumSignatureMissing      = errors.New("missing Column-Signature header")
	ErrWebhookConfigInvalid       = errors.New("webhook config is invalid")
	ErrWebhookTypeUnknown         = errors.New("webhook type is not supported")
	ErrWebhookConfigSecretMissing = errors.New("webhook config secret missing")
	ErrWebhookRequestFailed       = errors.New("failed to create webhooks request")
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
