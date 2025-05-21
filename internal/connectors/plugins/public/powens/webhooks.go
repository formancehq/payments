package powens

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
)

const (
	webhookSecretMetadataKey = "secret"
)

type supportedWebhook struct {
	urlPath string
	fn      func(context.Context, models.TranslateWebhookRequest) ([]models.WebhookResponse, error)
}

// TODO(polo): compression
func (p *Plugin) initWebhookConfig() {
	p.supportedWebhooks = map[client.WebhookEventType]supportedWebhook{
		client.WebhookEventTypeUserCreated: {
			urlPath: "/user-created",
			fn:      p.handleUserCreated,
		},
		client.WebhookEventTypeUserDeleted: {
			urlPath: "/user-deleted",
			fn:      p.handleUserDeleted,
		},
		client.WebhookEventTypeConnectionSynced: {
			urlPath: "/connection-synced",
			fn:      p.handleConnectionSynced,
		},
		client.WebhookEventTypeConnectionDeleted: {
			urlPath: "/connection-deleted",
			fn:      p.handleConnectionDeleted,
		},
		client.WebhookEventTypeAccountsFetched: {
			urlPath: "/accounts-fetched",
			fn:      p.handleAccountsFetched,
		},
		client.WebhookEventTypeAccountSynced: {
			urlPath: "/account-synced",
			fn:      p.handleAccountSynced,
		},
		client.WebhookEventTypeAccountDisabled: {
			urlPath: "/account-disabled",
			fn:      p.handleAccountDisabled,
		},
		client.WebhookEventTypeAccountEnabled: {
			urlPath: "/account-enabled",
			fn:      p.handleAccountEnabled,
		},
		client.WebhookEventTypeAccountFound: {
			urlPath: "/account-found",
			fn:      p.handleAccountFound,
		},
		client.WebhookEventTypeAccountOwnerhipsFound: {
			urlPath: "/account-ownerhips-found",
			fn:      p.handleAccountOwnerhipsFound,
		},
		client.WebhookEventTypeAccountCategorized: {
			urlPath: "/account-categorized",
			fn:      p.handleAccountCategorized,
		},
		client.WebhookEventTypeSubscriptionFound: {
			urlPath: "/subscription-found",
			fn:      p.handleSubscriptionFound,
		},
		client.WebhookEventTypeSubscriptionSynced: {
			urlPath: "/subscription-synced",
			fn:      p.handleSubscriptionSynced,
		},
		client.WebhookEventTypePaymentStateUpdated: {
			urlPath: "/payment-state-updated",
			fn:      p.handlePaymentStateUpdated,
		},
		client.WebhookEventTypeTransactionAttachmentsFound: {
			urlPath: "/transaction-attachments-found",
			fn:      p.handleTransactionAttachmentsFound,
		},
	}
}

func (p *Plugin) createWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	secretKey, err := p.client.CreateWebhookAuth(ctx, req.ConnectorID)
	if err != nil {
		return models.CreateWebhooksResponse{}, err
	}

	configs := make([]models.PSPWebhookConfig, 0, len(p.supportedWebhooks))
	for eventType, w := range p.supportedWebhooks {
		configs = append(configs, models.PSPWebhookConfig{
			Name:    string(eventType),
			URLPath: w.urlPath,
			Metadata: map[string]string{
				webhookSecretMetadataKey: secretKey,
			},
		})
	}

	return models.CreateWebhooksResponse{
		Configs: configs,
	}, nil
}

func (p *Plugin) verifyWebhook(_ context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	signatureDate, ok := req.Webhook.Headers["BI-Signature-Date"]
	if !ok || len(signatureDate) != 1 {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing powens signature date header: %w", models.ErrWebhookVerification)
	}

	signatureB64, ok := req.Webhook.Headers["BI-Signature"]
	if !ok || len(signatureB64) != 1 {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing powens signature header: %w", models.ErrWebhookVerification)
	}

	signature, err := base64.StdEncoding.DecodeString(signatureB64[0])
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid powens signature header: %w", models.ErrWebhookVerification)
	}

	messageToSign := fmt.Sprintf("POST.%s.%s.%s", req.Config.FullURL, signatureDate[0], req.Webhook.Body)

	secretKey, ok := req.Config.Metadata[webhookSecretMetadataKey]
	if !ok {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing powens secret key: %w", models.ErrWebhookVerification)
	}

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(messageToSign))
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(signature, expectedSignature) {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid powens signature: %w", models.ErrWebhookVerification)
	}

	return models.VerifyWebhookResponse{}, nil
}

func (p *Plugin) handleUserCreated(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleUserDeleted(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleConnectionSynced(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleConnectionDeleted(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleAccountsFetched(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleAccountSynced(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleAccountDisabled(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleAccountEnabled(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleAccountFound(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleAccountOwnerhipsFound(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleAccountCategorized(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleSubscriptionFound(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleSubscriptionSynced(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handlePaymentStateUpdated(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}

func (p *Plugin) handleTransactionAttachmentsFound(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	return nil, nil
}
