package client

import (
	"errors"
	"fmt"
)

var (
	ErrWebhookUrlMissing              = errors.New("webhook url is not set")
	ErrMissingSelectedEventCategory   = errors.New("selected_event_category is not set in fromPayload")
	ErrWebhookSharedSecretMissing     = errors.New("webhook shared secret is not set")
	ErrWebhookHeaderXSignatureMissing = errors.New("missing X-Signature-Sha256 header")
	ErrWebhookNameUnknown             = errors.New("unknown webhook name")
	ErrWebhookRequestFailed           = errors.New("failed to create webhooks request")
)

type increaseError struct {
	Status int    `json:"status"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Errors []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"errors"`
}

func (ie *increaseError) Error() error {
	var err error
	if ie.Detail == "" {
		err = fmt.Errorf("unexpected status code: %d", ie.Status)
	} else {
		err = fmt.Errorf("%d: %s", ie.Status, ie.Detail)
	}

	return err
}
