package wise

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/models"
)

type supportedWebhook struct {
	triggerOn string
	urlPath   string
	fn        func(context.Context, models.TranslateWebhookRequest) (models.WebhookResponse, error)
	version   string
}

func (p *Plugin) createWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	var from client.Profile
	if req.FromPayload == nil {
		return models.CreateWebhooksResponse{}, models.ErrMissingFromPayloadInRequest
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.CreateWebhooksResponse{}, err
	}

	if req.WebhookBaseUrl == "" {
		return models.CreateWebhooksResponse{}, ErrStackPublicUrlMissing
	}

	configs := make([]models.PSPWebhookConfig, 0, len(p.supportedWebhooks))
	others := make([]models.PSPOther, 0, len(p.supportedWebhooks))
	for name, config := range p.supportedWebhooks {
		url, err := url.JoinPath(req.WebhookBaseUrl, config.urlPath)
		if err != nil {
			return models.CreateWebhooksResponse{}, err
		}

		resp, err := p.client.CreateWebhook(ctx, from.ID, name, config.triggerOn, url, config.version)
		if err != nil {
			return models.CreateWebhooksResponse{}, err
		}
		configs = append(configs, models.PSPWebhookConfig{
			Name:    name,
			URLPath: config.urlPath,
		})

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
		Configs: configs,
		Others:  others,
	}, nil
}

func (p *Plugin) translateTransferStateChangedWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.WebhookResponse, error) {
	transfer, err := p.client.TranslateTransferStateChangedWebhook(ctx, req.Webhook.Body)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	payment, err := fromTransferToPayment(transfer)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	return models.WebhookResponse{
		Payment: &payment,
	}, nil
}

func (p *Plugin) translateBalanceUpdateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.WebhookResponse, error) {
	update, err := p.client.TranslateBalanceUpdateWebhook(ctx, req.Webhook.Body)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	raw, err := json.Marshal(update)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	occuredAt, err := time.Parse(time.RFC3339, update.Data.OccurredAt)
	if err != nil {
		return models.WebhookResponse{}, fmt.Errorf("failed to parse created time: %w", err)
	}

	var paymentType models.PaymentType
	if update.Data.TransactionType == "credit" {
		paymentType = models.PAYMENT_TYPE_PAYIN
	} else {
		paymentType = models.PAYMENT_TYPE_PAYOUT
	}

	precision, ok := supportedCurrenciesWithDecimal[update.Data.Currency]
	if !ok {
		return models.WebhookResponse{}, nil
	}

	amount, err := currency.GetAmountWithPrecisionFromString(update.Data.Amount.String(), precision)
	if err != nil {
		return models.WebhookResponse{}, err
	}

	payment := models.PSPPayment{
		Reference: update.Data.TransferReference,
		CreatedAt: occuredAt,
		Type:      paymentType,
		Amount:    amount,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, update.Data.Currency),
		Scheme:    models.PAYMENT_SCHEME_OTHER,
		Status:    models.PAYMENT_STATUS_SUCCEEDED,
		Raw:       raw,
	}

	switch paymentType {
	case models.PAYMENT_TYPE_PAYIN:
		payment.SourceAccountReference = pointer.For(fmt.Sprintf("%d", update.Data.BalanceID))
	case models.PAYMENT_TYPE_PAYOUT:
		payment.DestinationAccountReference = pointer.For(fmt.Sprintf("%d", update.Data.BalanceID))
	}

	return models.WebhookResponse{
		Payment: &payment,
	}, nil
}

func (p *Plugin) verifySignature(body []byte, signature string) error {
	msgHash := sha256.New()
	_, err := msgHash.Write(body)
	if err != nil {
		return err
	}
	msgHashSum := msgHash.Sum(nil)

	data, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature for wise webhook: %w", err)
	}

	err = rsa.VerifyPKCS1v15(p.config.webhookPublicKey, crypto.SHA256, msgHashSum, data)
	if err != nil {
		return err
	}

	return nil
}
