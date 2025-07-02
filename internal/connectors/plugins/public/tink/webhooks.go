package tink

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
)

// https://docs.tink.com/resources/transactions/introduction-to-transactions
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
		client.AccountTransactionsDeleted: {
			urlPath: "/account-transactions-deleted",
			fn:      p.handleAccountTransactionsDeleted,
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
	}
}

func (p *Plugin) createWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	if req.ConnectorID == "" {
		return models.CreateWebhooksResponse{}, fmt.Errorf("missing connector ID: %w", models.ErrInvalidRequest)
	}
	if req.WebhookBaseUrl == "" {
		return models.CreateWebhooksResponse{}, fmt.Errorf("missing webhook base URL: %w", models.ErrInvalidRequest)
	}

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

func (p *Plugin) deleteWebhooks(ctx context.Context, req models.UninstallRequest) error {
	for _, config := range req.WebhookConfigs {
		if config.Metadata == nil ||
			config.Metadata[webhookIDMetadataKey] == "" {
			continue
		}

		err := p.client.DeleteWebhook(ctx, config.Metadata[webhookIDMetadataKey])
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Plugin) verifyWebhook(_ context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	if req.Config == nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing webhook config: %w", models.ErrWebhookVerification)
	}

	if req.Webhook.Headers == nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing webhook: %w", models.ErrWebhookVerification)
	}

	header, ok := req.Webhook.Headers["X-Tink-Signature"]
	if !ok || len(header) != 1 {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing tink signature header: %w", models.ErrWebhookVerification)
	}

	timestamp, signature, err := splitVerificationHeader(header[0])
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid tink signature header %s: %w", header[0], models.ErrWebhookVerification)
	}

	timestampInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	createdAt := time.Unix(timestampInt, 0)
	if time.Since(createdAt) > webhookCreatedAtThreshold {
		return models.VerifyWebhookResponse{}, fmt.Errorf("webhook created at %s is too old: %w", createdAt, models.ErrWebhookVerification)
	}

	messageToSign := fmt.Sprintf("%s.%s", timestamp, string(req.Webhook.Body))
	if req.Config.Metadata == nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("missing webhook config: %w", models.ErrWebhookVerification)
	}

	secret := req.Config.Metadata[webhookSecretMetadataKey]
	if secret == "" {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid signature: %w", models.ErrWebhookVerification)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(messageToSign))
	computedSignature := hex.EncodeToString(mac.Sum(nil))

	if computedSignature != signature {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid signature: %w", models.ErrWebhookVerification)
	}

	return models.VerifyWebhookResponse{}, nil
}

func splitVerificationHeader(header string) (string, string, error) {
	parts := strings.Split(header, ",")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid tink signature header %s: %w", header, models.ErrWebhookVerification)
	}

	timestampPart := strings.TrimSpace(parts[0])
	if !strings.HasPrefix(timestampPart, "t=") {
		return "", "", fmt.Errorf("invalid tink signature header %s: %w", header, models.ErrWebhookVerification)
	}
	timestamp := strings.TrimPrefix(timestampPart, "t=")

	signaturePart := strings.TrimSpace(parts[1])
	if !strings.HasPrefix(signaturePart, "v1=") {
		return "", "", fmt.Errorf("invalid tink signature header %s: %w", header, models.ErrWebhookVerification)
	}
	signature := strings.TrimPrefix(signaturePart, "v1=")

	return timestamp, signature, nil
}

func (p *Plugin) handleAccountBookedTransactionsModified(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	// This event is fired when an account has new or updated transactions that
	// have status BOOKED.

	// Note: We don't need to do anything here as we will receive the
	// handle AccountTransactionsModified webhook

	fmt.Println("account booked transactions modified", string(req.Webhook.Body))

	return nil, nil
}

// https://docs.tink.com/resources/transactions/webhooks-for-transactions#event-account-transactions-modified
func (p *Plugin) handleAccountTransactionsModified(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	// This event is fired when an account has new or updated transactions,
	// regardless of their booking status.
	// https://docs.tink.com/resources/transactions/webhooks-for-transactions#event-account-transactions-modified

	accountTransactionsModifiedWebhook, err := p.client.GetAccountTransactionsModifiedWebhook(ctx, req.Webhook.Body)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(accountTransactionsModifiedWebhook)
	if err != nil {
		return nil, err
	}

	response := models.WebhookResponse{
		TransactionReadyToFetch: &models.TransactionReadyToFetch{
			FromPayload: payload,
		},
	}

	return []models.WebhookResponse{response}, nil
}

func (p *Plugin) handleAccountTransactionsDeleted(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	// This event is triggered when an account has deleted transactions,
	// regardless of their booking status.
	// https://docs.tink.com/resources/transactions/webhooks-for-transactions#event-account-transactions-deleted

	// Note: launch a deletion of transactions -> // TODO(polo): add a new response to call the delete_payments workflow

	fmt.Println("account transactions deleted", string(req.Webhook.Body))

	return nil, nil
}

func (p *Plugin) handleAccountCreated(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	// https://docs.tink.com/entries/articles/event-account-created

	// Note: Nothing to do here for now.

	accountCreatedWebhook, err := p.client.GetAccountCreatedWebhook(ctx, req.Webhook.Body)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(accountCreatedWebhook)
	if err != nil {
		return nil, err
	}

	response := models.WebhookResponse{
		TransactionReadyToFetch: &models.TransactionReadyToFetch{
			FromPayload: payload,
		},
	}

	return []models.WebhookResponse{response}, nil
}

func (p *Plugin) handleAccountUpdated(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	// https://docs.tink.com/entries/articles/event-account-updated

	fmt.Println("account updated", string(req.Webhook.Body))

	return nil, nil
}

func (p *Plugin) handleRefreshFinished(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	// The refresh:finished event is triggered when a refresh operation has
	// finished for a credentials object. This occurs for both on-demand and
	// background refreshes.
	// The event is triggered if the refresh attempt was successsful or
	// unsuccessful. In the case of an unsuccessful refresh, the type of error
	// is specified in the event.
	// This event is only triggered by an attempted refresh. For example, it
	// will not be trigged for refreshes that have been rate limited. For more
	// information, see rate limits.
	// https://docs.tink.com/resources/transactions/webhooks-for-transactions#event-refresh-finished

	fmt.Println("refresh finished", string(req.Webhook.Body))

	return nil, nil
}
