package increase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/formancehq/payments/pkg/connectors/increase/client"
	"github.com/formancehq/payments/pkg/connector"
)

const (
	// HeadersSignature is Increase webhook signature
	HeadersSignature               = "Increase-Webhook-Signature"
	eventSubscriptionStatusDeleted = "deleted"
	signatureScheme                = "v1"
	toleranceDuration              = 5 * time.Minute // 5-minute tolerance for timestamp validation
)

type supportedWebhook struct {
	urlPath string
	fn      func(context.Context, client.WebhookEvent) (connector.WebhookResponse, error)
}

func (p *Plugin) initWebhookConfig() map[client.EventCategory]supportedWebhook {
	p.supportedWebhooks = map[client.EventCategory]supportedWebhook{
		client.EventCategoryDeclinedTransactionCreated: {
			urlPath: "/declined_transaction/created",
			fn:      p.translateDeclinedTransaction,
		},
		client.EventCategoryTransactionCreated: {
			urlPath: "/transaction/created",
			fn:      p.translateTransaction,
		},
		client.EventCategoryPendingTransactionCreated: {
			urlPath: "/pending_transaction/created",
			fn:      p.translatePendingTransaction,
		},
		client.EventCategoryPendingTransactionUpdated: {
			urlPath: "/pending_transactions/updated",
			fn:      p.translatePendingTransaction,
		},
	}

	return p.supportedWebhooks
}

func (p *Plugin) createWebhooks(ctx context.Context, req connector.CreateWebhooksRequest) (connector.CreateWebhooksResponse, error) {
	var (
		from   client.CreateEventSubscriptionRequest
		others []connector.PSPOther
	)
	if req.FromPayload == nil {
		return connector.CreateWebhooksResponse{}, connector.ErrMissingFromPayloadInRequest
	}
	if req.WebhookBaseUrl == "" {
		return connector.CreateWebhooksResponse{}, client.ErrWebhookUrlMissing
	}

	if !strings.HasPrefix(req.WebhookBaseUrl, "https://") {
		return connector.CreateWebhooksResponse{}, fmt.Errorf("webhook URL must use HTTPS protocol")
	}

	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return connector.CreateWebhooksResponse{}, err
	}

	configs := make([]connector.PSPWebhookConfig, 0, len(p.supportedWebhooks))
	for eventType, config := range p.supportedWebhooks {
		url, err := url.JoinPath(req.WebhookBaseUrl, config.urlPath)
		if err != nil {
			return connector.CreateWebhooksResponse{}, err
		}

		from.URL = url
		from.SelectedEventCategory = string(eventType)
		idempotencyKey := p.generateIdempotencyKey(from.SelectedEventCategory, req.ConnectorID)
		resp, err := p.client.CreateEventSubscription(ctx, &from, idempotencyKey)
		if err != nil {
			return connector.CreateWebhooksResponse{}, err
		}

		configs = append(configs, connector.PSPWebhookConfig{
			Name:     string(eventType),
			URLPath:  config.urlPath,
			Metadata: map[string]string{"secret": p.webhookSharedSecret},
		})

		raw, err := json.Marshal(resp)
		if err != nil {
			return connector.CreateWebhooksResponse{}, err
		}

		others = append(others, connector.PSPOther{
			ID:    resp.ID,
			Other: raw,
		})
	}

	return connector.CreateWebhooksResponse{
		Others:  others,
		Configs: configs,
	}, nil
}

func (p *Plugin) translateWebhook(ctx context.Context, req connector.TranslateWebhookRequest) (connector.TranslateWebhookResponse, error) {
	config, ok := p.supportedWebhooks[client.EventCategory(req.Name)]
	if !ok {
		return connector.TranslateWebhookResponse{}, client.ErrWebhookNameUnknown
	}

	var webhook client.WebhookEvent
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return connector.TranslateWebhookResponse{}, fmt.Errorf("failed to unmarshal webhook: %w", err)
	}

	res, err := config.fn(ctx, webhook)
	if err != nil {
		return connector.TranslateWebhookResponse{}, err
	}

	return connector.TranslateWebhookResponse{
		Responses: []connector.WebhookResponse{res},
	}, nil
}

func (p *Plugin) translatePendingTransaction(ctx context.Context, webhook client.WebhookEvent) (connector.WebhookResponse, error) {
	var response connector.WebhookResponse
	transaction, err := p.client.GetPendingTransaction(ctx, webhook.AssociatedObjectID)
	if err != nil {
		return connector.WebhookResponse{}, err
	}

	pspPayment, err := p.mapPayment(transaction, connector.PAYMENT_STATUS_PENDING)
	if err != nil {
		return connector.WebhookResponse{}, fmt.Errorf("failed to map pending transaction payment: %w", err)
	}
	response.Payment = &pspPayment

	return response, nil
}

func (p *Plugin) translateTransaction(ctx context.Context, webhook client.WebhookEvent) (connector.WebhookResponse, error) {
	var response connector.WebhookResponse
	transaction, err := p.client.GetTransaction(ctx, webhook.AssociatedObjectID)
	if err != nil {
		return connector.WebhookResponse{}, err
	}

	pspPayment, err := p.mapPayment(transaction, connector.PAYMENT_STATUS_SUCCEEDED)
	if err != nil {
		return connector.WebhookResponse{}, fmt.Errorf("failed to map transaction payment: %w", err)
	}
	response.Payment = &pspPayment

	return response, nil
}

func (p *Plugin) translateDeclinedTransaction(ctx context.Context, webhook client.WebhookEvent) (connector.WebhookResponse, error) {
	var response connector.WebhookResponse
	transaction, err := p.client.GetDeclinedTransaction(ctx, webhook.AssociatedObjectID)
	if err != nil {
		return connector.WebhookResponse{}, err
	}

	pspPayment, err := p.mapPayment(transaction, connector.PAYMENT_STATUS_FAILED)
	if err != nil {
		return connector.WebhookResponse{}, fmt.Errorf("failed to map declined transaction payment: %w", err)
	}
	response.Payment = &pspPayment

	return response, nil
}

func (p *Plugin) verifyWebhookSignature(payload []byte, header string) error {
	timestamp, signatures, err := extractSignatureData(header)
	if err != nil {
		return err
	}

	signedPayload := fmt.Sprintf("%s.%s", timestamp, string(payload))
	expectedSignature, err := computeHMACSHA256(signedPayload, p.webhookSharedSecret)
	if err != nil {
		return err
	}

	if !validateTimestamp(timestamp) {
		return errors.New("timestamp outside tolerance window")
	}

	if !compareSignatures(expectedSignature, signatures) {
		return errors.New("invalid webhook signature")
	}

	return nil
}

func extractSignatureData(header string) (string, []string, error) {
	parts := strings.Split(header, ",")
	var timestamp string
	var signatures []string

	for _, part := range parts {
		pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(pair) != 2 {
			continue
		}
		key, value := pair[0], pair[1]
		switch key {
		case "t":
			timestamp = value
		case "v1":
			signatures = append(signatures, value)
		}
	}

	if timestamp == "" || len(signatures) == 0 {
		return "", nil, fmt.Errorf("invalid signature header")
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
	expectedSigBytes, err := hex.DecodeString(expectedSignature)
	if err != nil {
		fmt.Printf("Error decoding expected signature: %v\n", err)
		return false
	}

	for _, sig := range signatures {
		sigBytes, err := hex.DecodeString(sig)
		if err != nil {
			fmt.Printf("Error decoding received signature: %v\n", err)
			continue
		}
		if hmac.Equal(expectedSigBytes, sigBytes) {
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
