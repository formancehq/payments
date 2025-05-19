package tink

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
)

const (
	webhookCreatedAtThreshold = 5 * time.Minute

	webhookIDMetadataKey     = "webhook_id"
	webhookSecretMetadataKey = "secret"
)

type supportedWebhook struct {
	urlPath string
	fn      func(context.Context, models.TranslateWebhookRequest) ([]models.WebhookResponse, error)
}

func (p *Plugin) initWebhookConfig() {
	p.supportedWebhooks = map[client.WebhookEventType]supportedWebhook{
		client.AccountTransactionsModified: {
			urlPath: "/account-transactions-modified",
			fn:      p.handleAccountTransactionsModified,
		},
		client.AccountBookedTransactionsModified: {
			urlPath: "/account-booked-transactions-modified",
			fn:      p.handleAccountBookedTransactionsModified,
		},
		client.AccountCreated: {
			urlPath: "/account-created",
			fn:      p.handleAccountCreated,
		},
		client.AccountUpdated: {
			urlPath: "/account-updated",
			fn:      p.handleAccountUpdated,
		},
		client.RefreshFinished: {
			urlPath: "/refresh-finished",
			fn:      p.handleRefreshFinished,
		},
		client.PaymentUpdated: {
			urlPath: "/payment-updated",
			fn:      p.handlePaymentUpdated,
		},
	}
}

func (p *Plugin) createWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	configs := make([]models.PSPWebhookConfig, 0, len(p.supportedWebhooks))
	for eventType, w := range p.supportedWebhooks {
		webhookURL, err := url.JoinPath(req.WebhookBaseUrl, w.urlPath)
		if err != nil {
			return models.CreateWebhooksResponse{}, fmt.Errorf("failed to join path: %w", err)
		}

		resp, err := p.client.CreateWebhook(ctx, eventType, req.ConnectorID, webhookURL)
		if err != nil {
			return models.CreateWebhooksResponse{}, fmt.Errorf("failed to create webhook: %w", err)
		}

		configs = append(configs, models.PSPWebhookConfig{
			Name:    string(eventType),
			URLPath: w.urlPath,
			Metadata: map[string]string{
				webhookIDMetadataKey:     resp.ID,
				webhookSecretMetadataKey: resp.Secret,
			},
		})
	}

	return models.CreateWebhooksResponse{
		Configs: configs,
	}, nil
}

func (p *Plugin) verifyWebhook(ctx context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	header, ok := req.Webhook.Headers["X-Tink-Signature"]
	if !ok || len(header) != 1 {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing tink signature header: %w", models.ErrWebhookVerification)
	}

	timestamp, signature, err := splitVerificationHeader(header[0])
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("failed to split verification header: %w", err)
	}

	timestampInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	createdAt := time.Unix(timestampInt, 0)
	if time.Since(createdAt) > webhookCreatedAtThreshold {
		return models.VerifyWebhookResponse{}, fmt.Errorf("webhook created at %s is too old: %w", createdAt, models.ErrWebhookVerification)
	}

	messageToSign := fmt.Sprintf("%s.%s", timestamp, req.Webhook.Body)
	secret := req.Config.Metadata[webhookIDMetadataKey]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(messageToSign))
	computedSignature := mac.Sum(nil)

	if !hmac.Equal(computedSignature, []byte(signature)) {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid signature: %w", models.ErrWebhookVerification)
	}

	return models.VerifyWebhookResponse{}, nil
}

func splitVerificationHeader(header string) (string, string, error) {
	parts := strings.Split(header, ",")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid tink signature header %s: %w", header, models.ErrWebhookVerification)
	}

	timestampPart := strings.Split(parts[0], "=")
	if len(timestampPart) != 2 || timestampPart[0] != "t" {
		return "", "", fmt.Errorf("invalid tink signature header %s: %w", header, models.ErrWebhookVerification)
	}

	timestamp := timestampPart[1]

	signaturePart := strings.Split(parts[1], "=")
	if len(signaturePart) != 2 || signaturePart[0] != "v1" {
		return "", "", fmt.Errorf("invalid tink signature header %s: %w", header, models.ErrWebhookVerification)
	}

	signature := signaturePart[1]

	return timestamp, signature, nil
}

func (p *Plugin) handleAccountTransactionsModified(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleAccountBookedTransactionsModified(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleAccountCreated(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleAccountUpdated(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleRefreshFinished(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handlePaymentUpdated(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}
