package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/webhook"
)

func (p *Plugin) createWebhooks(ctx context.Context, req models.CreateWebhooksRequest) ([]models.PSPWebhookConfig, error) {
	result, err := p.client.CreateWebhookEndpoint(ctx, req.WebhookBaseUrl)
	if err != nil {
		return nil, err
	}

	urlPath := strings.TrimPrefix(result.URL, req.WebhookBaseUrl)
	configs := []models.PSPWebhookConfig{
		{
			Name:    result.ID,
			URLPath: urlPath,
			Metadata: map[string]string{
				"secret":         result.Secret,
				"enabled_events": strings.Join(result.EnabledEvents, ","),
			},
		},
	}
	return configs, nil
}

func (p *Plugin) extractWebhookEvent(config *models.WebhookConfig, wh models.PSPWebhook) (evt stripe.Event, err error) {
	if config == nil || config.Metadata == nil {
		return evt, fmt.Errorf("config metadata missing for this webhook: %w", models.ErrWebhookVerification)
	}

	secret, ok := config.Metadata["secret"]
	if !ok {
		return evt, fmt.Errorf("secret missing from config: %w", models.ErrWebhookVerification)
	}

	payload := wh.Body
	headers, ok := wh.Headers["Stripe-Signature"]
	if !ok || len(headers) != 1 {
		return evt, fmt.Errorf("stripe signature header not found: %w", models.ErrWebhookVerification)
	}

	// Pass the request body and Stripe-Signature header to ConstructEvent, along
	// with the webhook signing key.
	event, err := webhook.ConstructEvent(payload, headers[0], secret)
	if err != nil {
		return evt, fmt.Errorf("error verifying webhook signature: %w", err)
	}
	return event, nil
}

func (p *Plugin) verifyWebhook(ctx context.Context, req models.VerifyWebhookRequest) (*string, error) {
	event, err := p.extractWebhookEvent(req.Config, req.Webhook)
	if err != nil {
		return nil, fmt.Errorf("error verifying webhook: %w", err)
	}
	return &event.ID, nil
}

func (p *Plugin) translateWebhook(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	event, err := p.extractWebhookEvent(req.Config, req.Webhook)
	if err != nil {
		return []models.WebhookResponse{}, fmt.Errorf("error translating webhook: %w", err)
	}

	var webhookResponses []models.WebhookResponse
	switch event.Type {
	case stripe.EventTypeBalanceAvailable:
		var balance stripe.Balance
		err := json.Unmarshal(event.Data.Raw, &balance)
		if err != nil {
			return []models.WebhookResponse{}, fmt.Errorf("failed to parse %q webhook JSON: %w", event.Type, err)
		}

		timestamp := time.Now().UTC()
		for _, a := range balance.Available {
			webhookResponses = append(webhookResponses, models.WebhookResponse{
				Balance: &models.PSPBalance{
					AccountReference: event.Account,
					Amount:           big.NewInt(a.Amount),
					Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, string(a.Currency)),
					CreatedAt:        timestamp,
				},
			})
		}
	default:
		return []models.WebhookResponse{}, fmt.Errorf("unhandled event type: %q", event.Type)
	}

	return webhookResponses, nil
}
