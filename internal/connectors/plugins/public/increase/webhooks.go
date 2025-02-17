package increase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

const (
	webhookTypeAccountCreated             = "account.created"
	webhookTypeDeclinedTransactionCreated = "declined_transaction.created"
	webhookTypePendingTransactionCreated  = "pending_transaction.created"
	webhookTypeTransactionCreated         = "transaction.created"
	webhookTypeExternalAccountCreated     = "external_account.created"
	webhookTypeAccountTransferCreated     = "account_transfer.created"
	webhookTypeCheckTransferCreated       = "check_transfer.created"
	webhookTypeWireTransferCreated        = "wire_transfer.created"
	webhookTypeRTPTransferCreated         = "real_time_payments_transfer.created"
	webhookTypACHTransferCreated          = "ach_transfer.created"
	HeadersSignature                      = "Increase-Webhook-Signature"
	eventSubscriptionStatusDeleted        = "deleted"
)

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

	return models.CreateWebhooksResponse{
		Others: others,
	}, nil
}

func (p *Plugin) translateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	signatures, ok := req.Webhook.Headers[HeadersSignature]
	if !ok || len(signatures) == 0 {
		return models.TranslateWebhookResponse{}, client.ErrWebhookHeaderXSignatureMissing
	}
	if err := p.client.VerifyWebhookSignature(req.Webhook.Body, signatures[0]); err != nil {
		return models.TranslateWebhookResponse{}, err
	}

	var webhook client.WebhookEvent
	if err := json.Unmarshal(req.Webhook.Body, &webhook); err != nil {
		return models.TranslateWebhookResponse{}, fmt.Errorf("failed to unmarshal webhook: %w", err)
	}

	var response models.WebhookResponse
	response.IdempotencyKey = webhook.ID

	switch webhook.Category {
	case webhookTypeAccountCreated:
		account, err := p.client.GetAccount(ctx, webhook.AssociatedObjectID)
		if err != nil {
			return models.TranslateWebhookResponse{}, err
		}

		pspAccount, err := p.mapAccount(account)
		if err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to map account: %w", err)
		}
		response.Account = &pspAccount

	case webhookTypeTransactionCreated:
		transaction, err := p.client.GetTransaction(ctx, webhook.AssociatedObjectID)
		if err != nil {
			return models.TranslateWebhookResponse{}, err
		}

		pspPayment, err := p.mapPayment(transaction, models.PAYMENT_STATUS_SUCCEEDED)
		if err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to map succeeded transaction payment: %w", err)
		}
		response.Payment = &pspPayment

	case webhookTypeDeclinedTransactionCreated:
		transaction, err := p.client.GetDeclinedTransaction(ctx, webhook.AssociatedObjectID)
		if err != nil {
			return models.TranslateWebhookResponse{}, err
		}

		pspPayment, err := p.mapPayment(transaction, models.PAYMENT_STATUS_FAILED)
		if err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to map declined transaction payment: %w", err)
		}
		response.Payment = &pspPayment

	case webhookTypePendingTransactionCreated:
		transaction, err := p.client.GetPendingTransaction(ctx, webhook.AssociatedObjectID)
		if err != nil {
			return models.TranslateWebhookResponse{}, err
		}

		pspPayment, err := p.mapPayment(transaction, models.PAYMENT_STATUS_PENDING)
		if err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to map pending transaction payment: %w", err)
		}
		response.Payment = &pspPayment

	case webhookTypeExternalAccountCreated:
		account, err := p.client.GetExternalAccount(ctx, webhook.AssociatedObjectID)
		if err != nil {
			return models.TranslateWebhookResponse{}, err
		}

		pspAccount, err := p.mapExternalAccount(account)
		if err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to map external account: %w", err)
		}
		response.Account = pspAccount

	case webhookTypeAccountTransferCreated:
		transfer, err := p.client.GetTransfer(ctx, webhook.AssociatedObjectID)
		if err != nil {
			return models.TranslateWebhookResponse{}, err
		}

		pspPayment, err := p.mapTransfer(transfer)
		if err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to map account transfer: %w", err)
		}
		response.Payment = pspPayment

	case webhookTypeCheckTransferCreated:
		transfer, err := p.client.GetCheckTransferPayout(ctx, webhook.AssociatedObjectID)
		if err != nil {
			return models.TranslateWebhookResponse{}, err
		}

		pspPayment, err := p.payoutToPayment(transfer)
		if err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to map check transfer: %w", err)
		}
		response.Payment = pspPayment

	case webhookTypeRTPTransferCreated:
		transfer, err := p.client.GetRTPTransferPayout(ctx, webhook.AssociatedObjectID)
		if err != nil {
			return models.TranslateWebhookResponse{}, err
		}

		pspPayment, err := p.payoutToPayment(transfer)
		if err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to map rtp transfer: %w", err)
		}
		response.Payment = pspPayment

	case webhookTypeWireTransferCreated:
		transfer, err := p.client.GetWireTransferPayout(ctx, webhook.AssociatedObjectID)
		if err != nil {
			return models.TranslateWebhookResponse{}, err
		}

		pspPayment, err := p.payoutToPayment(transfer)
		if err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to map wire transfer: %w", err)
		}
		response.Payment = pspPayment

	case webhookTypACHTransferCreated:
		transfer, err := p.client.GetACHTransferPayout(ctx, webhook.AssociatedObjectID)
		if err != nil {
			return models.TranslateWebhookResponse{}, err
		}

		pspPayment, err := p.payoutToPayment(transfer)
		if err != nil {
			return models.TranslateWebhookResponse{}, fmt.Errorf("failed to map ach transfer: %w", err)
		}
		response.Payment = pspPayment
	}

	return models.TranslateWebhookResponse{
		Responses: []models.WebhookResponse{response},
	}, nil
}
