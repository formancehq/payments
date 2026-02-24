package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/formancehq/payments/pkg/connector"
)

type EventType string

const (
	// Transfer
	EventTypeTransferNormalCreated   EventType = "TRANSFER_NORMAL_CREATED"
	EventTypeTransferNormalFailed    EventType = "TRANSFER_NORMAL_FAILED"
	EventTypeTransferNormalSucceeded EventType = "TRANSFER_NORMAL_SUCCEEDED"

	// PayOut
	EventTypePayoutNormalCreated    EventType = "PAYOUT_NORMAL_CREATED"
	EventTypePayoutNormalFailed     EventType = "PAYOUT_NORMAL_FAILED"
	EventTypePayoutNormalSucceeded  EventType = "PAYOUT_NORMAL_SUCCEEDED"
	EventTypePayoutInstantFailed    EventType = "INSTANT_PAYOUT_FAILED"
	EventTypePayoutInstantSucceeded EventType = "INSTANT_PAYOUT_SUCCEEDED"

	// PayIn
	EventTypePayinNormalCreated   EventType = "PAYIN_NORMAL_CREATED"
	EventTypePayinNormalFailed    EventType = "PAYIN_NORMAL_FAILED"
	EventTypePayinNormalSucceeded EventType = "PAYIN_NORMAL_SUCCEEDED"

	// Refund
	EventTypeTransferRefundCreated   EventType = "TRANSFER_REFUND_CREATED"
	EventTypeTransferRefundFailed    EventType = "TRANSFER_REFUND_FAILED"
	EventTypeTransferRefundSucceeded EventType = "TRANSFER_REFUND_SUCCEEDED"
	EventTypePayinRefundCreated      EventType = "PAYIN_REFUND_CREATED"
	EventTypePayinRefundFailed       EventType = "PAYIN_REFUND_FAILED"
	EventTypePayinRefundSucceeded    EventType = "PAYIN_REFUND_SUCCEEDED"
	EventTypePayOutRefundCreated     EventType = "PAYOUT_REFUND_CREATED"
	EventTypePayOutRefundFailed      EventType = "PAYOUT_REFUND_FAILED"
	EventTypePayOutRefundSucceeded   EventType = "PAYOUT_REFUND_SUCCEEDED"
)

var (
	AllEventTypes = []EventType{
		EventTypeTransferNormalCreated,
		EventTypeTransferNormalFailed,
		EventTypeTransferNormalSucceeded,
		EventTypePayoutNormalCreated,
		EventTypePayoutNormalFailed,
		EventTypePayoutNormalSucceeded,
		EventTypePayoutInstantFailed,
		EventTypePayoutInstantSucceeded,
		EventTypePayinNormalCreated,
		EventTypePayinNormalFailed,
		EventTypePayinNormalSucceeded,
		EventTypeTransferRefundCreated,
		EventTypeTransferRefundFailed,
		EventTypeTransferRefundSucceeded,
		EventTypePayinRefundCreated,
		EventTypePayinRefundFailed,
		EventTypePayinRefundSucceeded,
		EventTypePayOutRefundCreated,
		EventTypePayOutRefundFailed,
		EventTypePayOutRefundSucceeded,
	}
)

type Webhook struct {
	ResourceID string    `json:"ResourceId"`
	Date       int64     `json:"Date"`
	EventType  EventType `json:"EventType"`
}

type Hook struct {
	ID        string    `json:"Id"`
	URL       string    `json:"Url"`
	Status    string    `json:"Status"`
	Validity  string    `json:"Validity"`
	EventType EventType `json:"EventType"`
}

func (c *client) ListAllHooks(ctx context.Context) ([]*Hook, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_hooks")

	endpoint := fmt.Sprintf("%s/v2.01/%s/hooks", c.endpoint, c.clientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create list hooks request: %w", err)
	}

	q := req.URL.Query()
	q.Add("per_page", "100") // Should be enough, since we're creating only a few
	q.Add("Sort", "CreationDate:ASC")
	req.URL.RawQuery = q.Encode()

	var hooks []*Hook
	statusCode, err := c.httpClient.Do(ctx, req, &hooks, nil)
	if err != nil {
		return nil, connector.NewWrappedError(
			fmt.Errorf("failed to list hooks: status code %d", statusCode),
			err,
		)
	}
	return hooks, nil
}

type CreateHookRequest struct {
	EventType EventType `json:"EventType"`
	URL       string    `json:"Url"`
}

func (c *client) CreateHook(ctx context.Context, eventType EventType, URL string) error {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_hook")

	body, err := json.Marshal(&CreateHookRequest{
		EventType: eventType,
		URL:       URL,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal create hook request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v2.01/%s/hooks", c.endpoint, c.clientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create hooks request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var errRes mangopayError
	_, err = c.httpClient.Do(ctx, req, nil, &errRes)
	if err != nil {
		return connector.NewWrappedError(
			fmt.Errorf("failed to create hook: %v", errRes.Error()),
			err,
		)
	}
	return nil
}

type UpdateHookRequest struct {
	URL    string `json:"Url"`
	Status string `json:"Status"`
}

func (c *client) UpdateHook(ctx context.Context, hookID string, URL string) error {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "update_hook")

	body, err := json.Marshal(&UpdateHookRequest{
		URL:    URL,
		Status: "ENABLED",
	})
	if err != nil {
		return fmt.Errorf("failed to marshal udpate hook request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v2.01/%s/hooks/%s", c.endpoint, c.clientID, hookID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create update hooks request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var errRes mangopayError
	_, err = c.httpClient.Do(ctx, req, nil, &errRes)
	if err != nil {
		return connector.NewWrappedError(
			fmt.Errorf("failed to update hook: %v", errRes.Error()),
			err,
		)
	}
	return nil
}
