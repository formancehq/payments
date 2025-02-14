package client

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type WebhookEvent struct {
	ID                   string    `json:"id"`
	Type                 string    `json:"type"`
	CreatedAt            time.Time `json:"created_at"`
	Category             string    `json:"category"`
	AssociatedObjectID   string    `json:"associated_object_id"`
	AssociatedObjectType string    `json:"associated_object_type"`
}

type EventSubscription struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

type UpdateEventSubscriptionRequest struct {
	Status string `json:"status"`
}

type CreateEventSubscriptionRequest struct {
	OauthConnectionID     string `json:"oauth_connection_id"`
	SelectedEventCategory string `json:"selected_event_category"`
	SharedSecret          string `json:"shared_secret"`
	URL                   string `json:"url"`
}

const (
	signatureScheme   = "v1"
	toleranceDuration = 5 * time.Minute // 5-minute tolerance for timestamp validation
)

func (c *client) CreateEventSubscription(ctx context.Context, es *CreateEventSubscriptionRequest) (*EventSubscription, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_event_subscription")

	if es.SharedSecret == "" {
		es.SharedSecret = c.webhookSharedSecret
	}

	body, err := json.Marshal(es)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "event_subscriptions", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create webhooks request: %w", err)
	}

	var res EventSubscription
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create web hooks: %w %w", err, errRes.Error())
	}
	return &res, nil
}

func (c *client) ListEventSubscriptions(ctx context.Context) ([]*EventSubscription, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_event_subscription")

	req, err := c.newRequest(ctx, http.MethodGet, "event_subscriptions", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhooks request: %w", err)
	}

	var res responseWrapper[[]*EventSubscription]
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to list web hooks: %w %w", err, errRes.Error())
	}
	return res.Data, nil
}

func (c *client) UpdateEventSubscription(ctx context.Context, es *UpdateEventSubscriptionRequest, eventID string) (*EventSubscription, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "update_event_subscription")

	body, err := json.Marshal(es)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPatch, fmt.Sprintf("event_subscriptions/%s", eventID), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create webhooks request: %w", err)
	}

	var res EventSubscription
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to update web hooks: %w %w", err, errRes.Error())
	}
	return &res, nil
}

func (c *client) VerifyWebhookSignature(payload []byte, header string) error {
	timestamp, signatures, err := extractSignatureData(header)
	if err != nil {
		return err
	}

	signedPayload := fmt.Sprintf("%s.%s", timestamp, payload)

	expectedSignature, err := computeHMACSHA256(signedPayload, c.webhookSharedSecret)
	if err != nil {
		return err
	}

	if !compareSignatures(expectedSignature, signatures) {
		return fmt.Errorf("invalid webhook signature: %w", err)
	}

	if !validateTimestamp(timestamp) {
		return errors.New("timestamp outside tolerance window")
	}

	return nil
}

func extractSignatureData(header string) (string, []string, error) {
	parts := strings.Split(header, ",")
	var timestamp string
	var signatures []string

	for _, part := range parts {
		pair := strings.SplitN(part, "=", 2)
		if len(pair) != 2 {
			continue
		}
		prefix, value := pair[0], pair[1]
		switch prefix {
		case "t":
			timestamp = value
		case signatureScheme:
			signatures = append(signatures, value)
		}
	}

	if timestamp == "" || len(signatures) == 0 {
		return "", nil, errors.New("invalid signature header")
	}
	return timestamp, signatures, nil
}

func computeHMACSHA256(message, secret string) (string, error) {
	mac := hmac.New(sha256.New, []byte(secret))
	_, err := mac.Write([]byte(message))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func compareSignatures(expectedSignature string, signatures []string) bool {
	for _, sig := range signatures {
		if hmac.Equal([]byte(expectedSignature), []byte(sig)) {
			return true
		}
	}
	return false
}

func validateTimestamp(timestamp string) bool {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return false
	}
	diff := time.Since(t)
	return diff <= toleranceDuration && diff >= -toleranceDuration
}
