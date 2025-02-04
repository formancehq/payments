package increase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

const (
	webhookTypeAccountCreated     = "account.created"
	webhookTypeTransactionCreated = "transaction.created"
	webhookTypeTransferCreated    = "transfer.created"
)

type webhookConfig struct {
	urlPath string
	fn      func(context.Context, webhookTranslateRequest) (models.WebhookResponse, error)
}

type webhookTranslateRequest struct {
	req     models.TranslateWebhookRequest
	webhook *client.WebhookEvent
}

func (p *Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	// Increase doesn't require webhook creation via API
	// Webhooks are configured in the dashboard
	return models.CreateWebhooksResponse{}, nil
}

func (p *Plugin) TranslateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	var webhook client.WebhookEvent
	if err := json.Unmarshal(req.Webhook.Raw, &webhook); err != nil {
		return models.TranslateWebhookResponse{}, fmt.Errorf("failed to unmarshal webhook: %w", err)
	}

	var response models.WebhookResponse
	response.IdempotencyKey = webhook.ID

	switch webhook.Type {
	case webhookTypeAccountCreated:
		var account client.Account
		if err := json.Unmarshal(webhook.Data, &account); err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to unmarshal account: %w", err)
		}

		raw, err := json.Marshal(account)
		if err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to marshal account: %w", err)
		}

		response.Account = &models.PSPAccount{
			ID:        account.ID,
			CreatedAt: account.CreatedAt,
			Reference: account.ID,
			Type:      models.AccountType(account.Type),
			Status:    models.AccountStatus(account.Status),
			Raw:       raw,
		}

	case webhookTypeTransactionCreated:
		var transaction client.Transaction
		if err := json.Unmarshal(webhook.Data, &transaction); err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to unmarshal transaction: %w", err)
		}

		raw, err := json.Marshal(transaction)
		if err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to marshal transaction: %w", err)
		}

		status := models.PaymentStatusSucceeded
		switch transaction.Status {
		case "pending":
			status = models.PaymentStatusPending
		case "declined":
			status = models.PaymentStatusFailed
		}

		response.Payment = &models.PSPPayment{
			ID:        transaction.ID,
			CreatedAt: transaction.CreatedAt,
			Reference: transaction.ID,
			Type:      models.PaymentType(transaction.Type),
			Status:    status,
			Amount:    transaction.Amount,
			Currency:  transaction.Currency,
			Raw:       raw,
		}

	case webhookTypeTransferCreated:
		var transfer client.Transfer
		if err := json.Unmarshal(webhook.Data, &transfer); err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to unmarshal transfer: %w", err)
		}

		raw, err := json.Marshal(transfer)
		if err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to marshal transfer: %w", err)
		}

		response.Payment = &models.PSPPayment{
			ID:        transfer.ID,
			CreatedAt: transfer.CreatedAt,
			Reference: transfer.ID,
			Type:      models.PaymentType(transfer.Type),
			Status:    models.PaymentStatus(transfer.Status),
			Amount:    transfer.Amount,
			Currency:  transfer.Currency,
			Raw:       raw,
		}
	}

	return models.TranslateWebhookResponse{
		Responses: []models.WebhookResponse{response},
	}, nil
}
