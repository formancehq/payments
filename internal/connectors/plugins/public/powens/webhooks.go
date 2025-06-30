package powens

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/powens/client"
	"github.com/formancehq/payments/internal/models"
)

const (
	webhookSecretMetadataKey = "secret"
)

type supportedWebhook struct {
	urlPath        string
	trimFunction   func(context.Context, models.TrimWebhookRequest) (models.TrimWebhookResponse, error)
	handleFunction func(context.Context, models.TranslateWebhookRequest) ([]models.WebhookResponse, error)
}

// TODO(polo): compression
func (p *Plugin) initWebhookConfig() {
	p.supportedWebhooks = map[client.WebhookEventType]supportedWebhook{
		client.WebhookEventTypeUserCreated: {
			urlPath:        "/user-created",
			handleFunction: p.handleUserCreated,
		},
		client.WebhookEventTypeUserDeleted: {
			urlPath:        "/user-deleted",
			handleFunction: p.handleUserDeleted,
		},
		client.WebhookEventTypeConnectionSynced: {
			urlPath:        "/connection-synced",
			handleFunction: p.handleConnectionSynced,
		},
		client.WebhookEventTypeConnectionDeleted: {
			urlPath:        "/connection-deleted",
			handleFunction: p.handleConnectionDeleted,
		},
		client.WebhookEventTypeAccountsFetched: {
			urlPath:        "/accounts-fetched",
			handleFunction: p.handleAccountsFetched,
		},
		client.WebhookEventTypeAccountSynced: {
			urlPath:        "/account-synced",
			trimFunction:   p.trimAccountsSynced,
			handleFunction: p.handleAccountSynced,
		},
		client.WebhookEventTypeAccountDisabled: {
			urlPath:        "/account-disabled",
			handleFunction: p.handleAccountDisabled,
		},
		client.WebhookEventTypeAccountEnabled: {
			urlPath:        "/account-enabled",
			handleFunction: p.handleAccountEnabled,
		},
		client.WebhookEventTypeAccountFound: {
			urlPath:        "/account-found",
			handleFunction: p.handleAccountFound,
		},
		client.WebhookEventTypeAccountOwnerhipsFound: {
			urlPath:        "/account-ownerhips-found",
			handleFunction: p.handleAccountOwnerhipsFound,
		},
		client.WebhookEventTypeAccountCategorized: {
			urlPath:        "/account-categorized",
			handleFunction: p.handleAccountCategorized,
		},
		client.WebhookEventTypeSubscriptionFound: {
			urlPath:        "/subscription-found",
			handleFunction: p.handleSubscriptionFound,
		},
		client.WebhookEventTypeSubscriptionSynced: {
			urlPath:        "/subscription-synced",
			handleFunction: p.handleSubscriptionSynced,
		},
		client.WebhookEventTypePaymentStateUpdated: {
			urlPath:        "/payment-state-updated",
			handleFunction: p.handlePaymentStateUpdated,
		},
		client.WebhookEventTypeTransactionAttachmentsFound: {
			urlPath:        "/transaction-attachments-found",
			handleFunction: p.handleTransactionAttachmentsFound,
		},
	}
}

func (p *Plugin) createWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	secretKey, err := p.client.CreateWebhookAuth(ctx, p.name)
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

func (p *Plugin) deleteWebhooks(ctx context.Context, req models.UninstallRequest) error {
	auths, err := p.client.ListWebhookAuths(ctx)
	if err != nil {
		return err
	}

	for _, auth := range auths {
		if auth.Name == p.name {
			if err := p.client.DeleteWebhookAuth(ctx, auth.ID); err != nil {
				return err
			}
			break
		}
	}

	return nil
}

func (p *Plugin) verifyWebhook(_ context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	signatureDate, ok := req.Webhook.Headers["Bi-Signature-Date"]
	if !ok || len(signatureDate) != 1 {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing powens signature date header: %w", models.ErrWebhookVerification)
	}

	signature, ok := req.Webhook.Headers["Bi-Signature"]
	if !ok || len(signature) != 1 {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing powens signature header: %w", models.ErrWebhookVerification)
	}

	u, err := url.Parse(req.Config.FullURL)
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid powens url: %w", models.ErrWebhookVerification)
	}

	secretKey, ok := req.Config.Metadata[webhookSecretMetadataKey]
	if !ok {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing powens secret key: %w", models.ErrWebhookVerification)
	}

	messageToSign := fmt.Sprintf("POST./%s.%s.%s", u.Path, signatureDate[0], string(req.Webhook.Body))
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(messageToSign))
	expectedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if expectedSignature != signature[0] {
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

func (p *Plugin) trimAccountsSynced(_ context.Context, req models.TrimWebhookRequest) (models.TrimWebhookResponse, error) {
	if len(req.Webhook.Body) == 0 {
		return models.TrimWebhookResponse{}, fmt.Errorf("missing powens accounts synced webhook body: %w", models.ErrValidation)
	}

	// Unmarshal then marshal again to remove all the other fields that we do
	// not need.
	var webhook client.AccountSyncedWebhook
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return models.TrimWebhookResponse{}, err
	}

	body, err := json.Marshal(&webhook)
	if err != nil {
		return models.TrimWebhookResponse{}, err
	}

	req.Webhook.Body = body

	return models.TrimWebhookResponse{
		Webhook: req.Webhook,
	}, nil
}

func (p *Plugin) handleAccountSynced(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	var webhook client.AccountSyncedWebhook
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return nil, err
	}

	return []models.WebhookResponse{
		{
			TransactionReadyToFetch: &models.TransactionReadyToFetch{
				ID:          pointer.For(strconv.Itoa(webhook.ConnectionID)),
				FromPayload: req.Webhook.Body,
			},
		},
	}, nil
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
