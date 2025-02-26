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

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

const (
	// HeadersSignature is Increase webhook signature
	HeadersSignature               = "Increase-Webhook-Signature"
	eventSubscriptionStatusDeleted = "deleted"
	signatureScheme                = "v1"
	toleranceDuration              = 5 * time.Minute // 5-minute tolerance for timestamp validation
)

type webhookConfig struct {
	urlPath string
	fn      func(context.Context, client.WebhookEvent) (models.WebhookResponse, error)
}

type defaultVerifier struct {
	webhookSharedSecret string
}

type WebhookVerifier interface {
	verifyWebhookSignature(payload []byte, header string) error
}

func (p *Plugin) initWebhookConfig() map[client.EventCategory]webhookConfig {
	p.webhookConfigs = map[client.EventCategory]webhookConfig{
		client.EventCategoryAccountCreated: {
			urlPath: "/accounts/created",
			fn:      p.translateAccount,
		},
		client.EventCategoryACHTransferCreated: {
			urlPath: "/ach_transfers/created",
			fn:      p.translateAchTransfer,
		},
		client.EventCategoryAccountTransferCreated: {
			urlPath: "/account_transfers/created",
			fn:      p.translateAccountTransfer,
		},
		client.EventCategoryCheckTransferCreated: {
			urlPath: "/check_transfers/created",
			fn:      p.translateCheckTransfer,
		},
		client.EventCategoryDeclinedTransactionCreated: {
			urlPath: "/declined_transactions/created",
			fn:      p.translateDeclinedTransaction,
		},
		client.EventCategoryExternalAccountCreated: {
			urlPath: "/external_accounts/created",
			fn:      p.translateExternalAccount,
		},
		client.EventCategoryPendingTransactionCreated: {
			urlPath: "/pending_transactions/created",
			fn:      p.translatePendingTransaction,
		},
		client.EventCategoryRTPTransferCreated: {
			urlPath: "/real_time_payments_transfers/created",
			fn:      p.translateRTPTransfer,
		},
		client.EventCategoryTransactionCreated: {
			urlPath: "/transactions/created",
			fn:      p.translateTransaction,
		},
		client.EventCategoryWireTransferCreated: {
			urlPath: "/wire_transfers/created",
			fn:      p.translateWireTransfer,
		},
	}

	return p.webhookConfigs
}

func (p *Plugin) createWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	var (
		from   client.CreateEventSubscriptionRequest
		others []models.PSPOther
	)
	if req.FromPayload == nil {
		return models.CreateWebhooksResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if req.WebhookBaseUrl == "" {
		return models.CreateWebhooksResponse{}, client.ErrWebhookUrlMissing
	}

	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.CreateWebhooksResponse{}, err
	}

	if from.SelectedEventCategory == "" {
		return models.CreateWebhooksResponse{}, client.ErrMissingSelectedEventCategory
	}

	for eventType, config := range p.webhookConfigs {
		url, err := url.JoinPath(req.WebhookBaseUrl, config.urlPath)
		if err != nil {
			return models.CreateWebhooksResponse{}, err
		}

		from.URL = url
		from.SelectedEventCategory = string(eventType)
		resp, err := p.client.CreateEventSubscription(ctx, &from)
		if err != nil {
			return models.CreateWebhooksResponse{}, fmt.Errorf("failed to create webhook subscription: %w", err)
		}

		raw, err := json.Marshal(resp)
		if err != nil {
			return models.CreateWebhooksResponse{}, err
		}

		others = append(others, models.PSPOther{
			ID:    resp.ID,
			Other: raw,
		})
	}

	return models.CreateWebhooksResponse{
		Others: others,
	}, nil
}

func (p *Plugin) translateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	signatures, ok := req.Webhook.Headers[HeadersSignature]
	if !ok || len(signatures) == 0 {
		return models.TranslateWebhookResponse{}, client.ErrWebhookHeaderXSignatureMissing
	}
	if err := p.verifier.verifyWebhookSignature(req.Webhook.Body, signatures[0]); err != nil {
		return models.TranslateWebhookResponse{}, err
	}

	config, ok := p.webhookConfigs[client.EventCategory(req.Name)]
	if !ok {
		return models.TranslateWebhookResponse{}, client.ErrWebhookNameUnknown
	}

	var webhook client.WebhookEvent
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return models.TranslateWebhookResponse{}, fmt.Errorf("failed to unmarshal webhook: %w", err)
	}

	res, err := config.fn(ctx, webhook)
	if err != nil {
		return models.TranslateWebhookResponse{}, err
	}

	res.IdempotencyKey = webhook.ID

	return models.TranslateWebhookResponse{
		Responses: []models.WebhookResponse{res},
	}, nil
}

func (p *Plugin) translateAccount(ctx context.Context, webhook client.WebhookEvent) (models.WebhookResponse, error) {
	var response models.WebhookResponse
	account, err := p.client.GetAccount(ctx, webhook.AssociatedObjectID)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	pspAccount, err := p.mapAccount(account)
	if err != nil {
		return models.WebhookResponse{}, fmt.Errorf("failed to map account: %w", err)
	}
	response.Account = &pspAccount

	return response, nil
}

func (p *Plugin) translateAccountTransfer(ctx context.Context, webhook client.WebhookEvent) (models.WebhookResponse, error) {
	var response models.WebhookResponse
	transfer, err := p.client.GetTransfer(ctx, webhook.AssociatedObjectID)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	pspPayment, err := p.transferToPayment(transfer)
	if err != nil {
		return models.WebhookResponse{}, fmt.Errorf("failed to map account transfer: %w", err)
	}
	response.Payment = pspPayment

	return response, nil
}

func (p *Plugin) translateAchTransfer(ctx context.Context, webhook client.WebhookEvent) (models.WebhookResponse, error) {
	var response models.WebhookResponse
	transfer, err := p.client.GetACHTransferPayout(ctx, webhook.AssociatedObjectID)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	pspPayment, err := p.payoutToPayment(transfer)
	if err != nil {
		return models.WebhookResponse{}, fmt.Errorf("failed to map ach transfer: %w", err)
	}
	response.Payment = pspPayment

	return response, nil
}

func (p *Plugin) translateCheckTransfer(ctx context.Context, webhook client.WebhookEvent) (models.WebhookResponse, error) {
	var response models.WebhookResponse
	transfer, err := p.client.GetCheckTransferPayout(ctx, webhook.AssociatedObjectID)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	pspPayment, err := p.payoutToPayment(transfer)
	if err != nil {
		return models.WebhookResponse{}, fmt.Errorf("failed to map check transfer: %w", err)
	}
	response.Payment = pspPayment

	return response, nil
}

func (p *Plugin) translateDeclinedTransaction(ctx context.Context, webhook client.WebhookEvent) (models.WebhookResponse, error) {
	var response models.WebhookResponse
	transaction, err := p.client.GetDeclinedTransaction(ctx, webhook.AssociatedObjectID)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	pspPayment, err := p.mapPayment(transaction, models.PAYMENT_STATUS_FAILED)
	if err != nil {
		return models.WebhookResponse{}, fmt.Errorf("failed to map declined transaction payment: %w", err)
	}
	response.Payment = &pspPayment

	return response, nil
}

func (p *Plugin) translateExternalAccount(ctx context.Context, webhook client.WebhookEvent) (models.WebhookResponse, error) {
	var response models.WebhookResponse
	account, err := p.client.GetExternalAccount(ctx, webhook.AssociatedObjectID)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	pspAccount, err := p.mapExternalAccount(account)
	if err != nil {
		return models.WebhookResponse{}, fmt.Errorf("failed to map external account: %w", err)
	}
	response.Account = pspAccount

	return response, nil
}

func (p *Plugin) translatePendingTransaction(ctx context.Context, webhook client.WebhookEvent) (models.WebhookResponse, error) {
	var response models.WebhookResponse
	transaction, err := p.client.GetPendingTransaction(ctx, webhook.AssociatedObjectID)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	pspPayment, err := p.mapPayment(transaction, models.PAYMENT_STATUS_PENDING)
	if err != nil {
		return models.WebhookResponse{}, fmt.Errorf("failed to map pending transaction payment: %w", err)
	}
	response.Payment = &pspPayment

	return response, nil
}

func (p *Plugin) translateRTPTransfer(ctx context.Context, webhook client.WebhookEvent) (models.WebhookResponse, error) {
	var response models.WebhookResponse
	transfer, err := p.client.GetRTPTransferPayout(ctx, webhook.AssociatedObjectID)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	pspPayment, err := p.payoutToPayment(transfer)
	if err != nil {
		return models.WebhookResponse{}, fmt.Errorf("failed to map rtp transfer: %w", err)
	}
	response.Payment = pspPayment

	return response, nil
}

func (p *Plugin) translateTransaction(ctx context.Context, webhook client.WebhookEvent) (models.WebhookResponse, error) {
	var response models.WebhookResponse
	transaction, err := p.client.GetTransaction(ctx, webhook.AssociatedObjectID)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	pspPayment, err := p.mapPayment(transaction, models.PAYMENT_STATUS_SUCCEEDED)
	if err != nil {
		return models.WebhookResponse{}, fmt.Errorf("failed to map succeeded transaction payment: %w", err)
	}
	response.Payment = &pspPayment

	return response, nil
}

func (p *Plugin) translateWireTransfer(ctx context.Context, webhook client.WebhookEvent) (models.WebhookResponse, error) {
	var response models.WebhookResponse
	transfer, err := p.client.GetWireTransferPayout(ctx, webhook.AssociatedObjectID)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	pspPayment, err := p.payoutToPayment(transfer)
	if err != nil {
		return models.WebhookResponse{}, fmt.Errorf("failed to map wire transfer: %w", err)
	}
	response.Payment = pspPayment

	return response, nil
}

func (v *defaultVerifier) verifyWebhookSignature(payload []byte, header string) error {
	timestamp, signatures, err := extractSignatureData(header)
	if err != nil {
		return err
	}

	signedPayload := fmt.Sprintf("%s.%s", timestamp, payload)
	expectedSignature, err := computeHMACSHA256(signedPayload, v.webhookSharedSecret)
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
