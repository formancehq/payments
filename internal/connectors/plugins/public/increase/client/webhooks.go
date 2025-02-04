package client

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v2/api"
)

type WebhookEvent struct {
	ID                  string          `json:"id"`
	Type                string          `json:"type"`
	CreatedAt           time.Time       `json:"created_at"`
	Category            string          `json:"category"`
	AssociatedObjectID  string          `json:"associated_object_id"`
	AssociatedObjectType string         `json:"associated_object_type"`
	Data                map[string]any  `json:"data"`
}

type EventSubscription struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

type CreateEventSubscriptionRequest struct {
	URL string `json:"url"`
}

func (c *client) CreateEventSubscription(ctx context.Context, req *CreateEventSubscriptionRequest) (*EventSubscription, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "create_event_subscription")

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, "/event_subscriptions", body)
	if err != nil {
		return nil, err
	}

	var subscription EventSubscription
	if err := c.do(httpReq, &subscription); err != nil {
		return nil, err
	}

	return &subscription, nil
}

func (c *client) VerifyWebhookSignature(payload []byte, header string) error {
	if header == "" {
		return fmt.Errorf("missing Increase-Webhook-Signature header")
	}

	parts := strings.Split(header, ",")
	if len(parts) < 2 {
		return fmt.Errorf("invalid signature header format")
	}

	var timestamp string
	var signature string
	for _, part := range parts {
		if strings.HasPrefix(part, "t=") {
			timestamp = strings.TrimPrefix(part, "t=")
		} else if strings.HasPrefix(part, "v1=") {
			signature = strings.TrimPrefix(part, "v1=")
		}
	}

	if timestamp == "" || signature == "" {
		return fmt.Errorf("missing timestamp or signature")
	}

	ts, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %w", err)
	}

	// Check if timestamp is within tolerance (5 minutes)
	if time.Since(ts) > 5*time.Minute {
		return fmt.Errorf("webhook timestamp too old")
	}

	expectedSignature := computeHMAC([]byte(timestamp+"."+string(payload)), []byte(c.apiKey))
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

func computeHMAC(message, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return hex.EncodeToString(mac.Sum(nil))
}
