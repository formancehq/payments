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
	"github.com/google/uuid"
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
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid tink signature header: %w", models.ErrWebhookVerification)
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

	expectedMAC, err := hex.DecodeString(signature)
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid signature encoding: %w", models.ErrWebhookVerification)
	}

	computedMAC := mac.Sum(nil)
	if !hmac.Equal(computedMAC, expectedMAC) {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid signature: %w", models.ErrWebhookVerification)
	}

	return models.VerifyWebhookResponse{}, nil
}

func splitVerificationHeader(header string) (string, string, error) {
	var timestamp, signature string
	for _, part := range strings.Split(header, ",") {
		p := strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(p, "t="):
			timestamp = strings.TrimPrefix(p, "t=")
		case strings.HasPrefix(p, "v1="):
			signature = strings.TrimPrefix(p, "v1=")
		}
	}
	if timestamp == "" || signature == "" {
		return "", "", fmt.Errorf("invalid tink signature header: %w", models.ErrWebhookVerification)
	}
	return timestamp, signature, nil
}

func (p *Plugin) handleAccountBookedTransactionsModified(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	// This event is fired when an account has new or updated transactions that
	// have status BOOKED.

	// Note: We don't need to do anything here as we will receive the
	// handle AccountTransactionsModified webhook

	return nil, nil
}

type fetchNextDataRequest struct {
	UserID                                string    `json:"userId"`
	ExternalUserID                        string    `json:"externalUserId"`
	AccountID                             string    `json:"accountId"`
	TransactionEarliestModifiedBookedDate time.Time `json:"transactionEarliestModifiedBookedDate"`
	TransactionLatestModifiedBookedDate   time.Time `json:"transactionLatestModifiedBookedDate"`
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

	payload, err := json.Marshal(fetchNextDataRequest{
		UserID:                                accountTransactionsModifiedWebhook.UserID,
		ExternalUserID:                        accountTransactionsModifiedWebhook.ExternalUserID,
		AccountID:                             accountTransactionsModifiedWebhook.Account.ID,
		TransactionEarliestModifiedBookedDate: accountTransactionsModifiedWebhook.Transactions.EarliestModifiedBookedDate,
		TransactionLatestModifiedBookedDate:   accountTransactionsModifiedWebhook.Transactions.LatestModifiedBookedDate,
	})
	if err != nil {
		return nil, err
	}

	psuID, err := uuid.Parse(accountTransactionsModifiedWebhook.ExternalUserID)
	if err != nil {
		return nil, err
	}

	response := models.WebhookResponse{
		DataReadyToFetch: &models.PSPDataReadyToFetch{
			PSUID:       &psuID,
			FromPayload: payload,
			DataToFetch: []models.OpenBankingDataToFetch{
				models.OpenBankingDataToFetchAccountsAndBalances,
				models.OpenBankingDataToFetchPayments,
			},
		},
	}

	return []models.WebhookResponse{response}, nil
}

func (p *Plugin) handleAccountTransactionsDeleted(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	// This event is triggered when an account has deleted transactions,
	// regardless of their booking status.
	// https://docs.tink.com/resources/transactions/webhooks-for-transactions#event-account-transactions-deleted

	accountTransactionsDeletedWebhook, err := p.client.GetAccountTransactionsDeletedWebhook(ctx, req.Webhook.Body)
	if err != nil {
		return nil, err
	}

	responses := make([]models.WebhookResponse, 0, len(accountTransactionsDeletedWebhook.Transactions.IDs))
	for _, transactionID := range accountTransactionsDeletedWebhook.Transactions.IDs {
		response := models.WebhookResponse{
			PaymentToCancel: &models.PSPPaymentsToCancel{
				Reference: transactionID,
			},
		}
		responses = append(responses, response)
	}

	return responses, nil
}

func (p *Plugin) handleAccountCreated(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	// https://docs.tink.com/entries/articles/event-account-created

	accountCreatedWebhook, err := p.client.GetAccountCreatedWebhook(ctx, req.Webhook.Body)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(fetchNextDataRequest{
		UserID:         accountCreatedWebhook.UserID,
		ExternalUserID: accountCreatedWebhook.ExternalUserID,
		AccountID:      accountCreatedWebhook.ID,
	})
	if err != nil {
		return nil, err
	}

	psuID, err := uuid.Parse(accountCreatedWebhook.ExternalUserID)
	if err != nil {
		return nil, err
	}

	return []models.WebhookResponse{
		{
			DataReadyToFetch: &models.PSPDataReadyToFetch{
				PSUID:       &psuID,
				FromPayload: payload,
				DataToFetch: []models.OpenBankingDataToFetch{
					models.OpenBankingDataToFetchAccountsAndBalances,
					models.OpenBankingDataToFetchPayments,
				},
			},
		},
	}, nil
}

func (p *Plugin) handleAccountUpdated(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	// https://docs.tink.com/entries/articles/event-account-updated

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

	refreshFinishedWebhook, err := p.client.GetRefreshFinishedWebhook(ctx, req.Webhook.Body)
	if err != nil {
		return nil, err
	}

	switch refreshFinishedWebhook.CredentialsStatus {
	case "UPDATED":
		return []models.WebhookResponse{
			{
				UserConnectionReconnected: &models.PSPUserConnectionReconnected{
					PSPUserID:    refreshFinishedWebhook.UserID,
					ConnectionID: refreshFinishedWebhook.CredentialsID,
					At:           time.Unix(0, refreshFinishedWebhook.Finished*int64(time.Millisecond)),
				},
			},
		}, nil
	case "TEMPORARY_ERROR", "AUTHENTICATION_ERROR", "SESSION_EXPIRED":
		reason := refreshFinishedWebhook.DetailedError.DisplayMessage
		if reason == "" {
			reason = fmt.Sprintf("%s: %s", refreshFinishedWebhook.DetailedError.Type, refreshFinishedWebhook.DetailedError.Details.Reason)
		}
		return []models.WebhookResponse{
			{
				UserConnectionDisconnected: &models.PSPUserConnectionDisconnected{
					ConnectionID: refreshFinishedWebhook.CredentialsID,
					ErrorType:    models.ConnectionDisconnectedErrorTypeUserActionNeeded,
					At:           time.Unix(0, refreshFinishedWebhook.Finished*int64(time.Millisecond)),
					Reason:       &reason,
				},
			},
		}, nil
	}

	return nil, nil
}
