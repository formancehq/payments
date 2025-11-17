package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/webhook"
)

var webhookRelatedAccountIDKey = "webhook_related_account_id"

func (p *Plugin) createWebhooks(ctx context.Context, req models.CreateWebhooksRequest) ([]models.PSPWebhookConfig, error) {
	results, err := p.client.CreateWebhookEndpoints(ctx, req.WebhookBaseUrl)
	if err != nil {
		return nil, err
	}

	configs := make([]models.PSPWebhookConfig, 0, len(results))
	for _, result := range results {
		urlPath := strings.TrimPrefix(result.URL, req.WebhookBaseUrl)

		metadata := map[string]string{
			"secret":         result.Secret,
			"enabled_events": strings.Join(result.EnabledEvents, ","),
		}

		// if it's not a StripeConnect enabled webhook let's embed the root account ID so we can associate events with it
		if !strings.Contains(urlPath, client.StripeConnectUrlPrefix) {
			metadata[webhookRelatedAccountIDKey] = p.client.GetRootAccountID()
		}
		configs = append(configs, models.PSPWebhookConfig{
			Name:     result.ID,
			URLPath:  urlPath,
			Metadata: metadata,
		})
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

	// an account reference is only present if it's a StripeConnect account
	accountReference := event.Account
	if accountReference == "" {
		p.logger.WithField("url", req.Config.URLPath).Infof("RELATED account reference is blank")
		accountID, ok := req.Config.Metadata[webhookRelatedAccountIDKey]
		if !ok {
			return []models.WebhookResponse{}, fmt.Errorf("failed to find root account: %+v", event)
		}
		accountReference = accountID
	}

	eventCreatedAt := time.Unix(event.Created, 0)
	switch event.Type {
	case stripe.EventTypeBalanceAvailable:
		var balance stripe.Balance
		err := json.Unmarshal(event.Data.Raw, &balance)
		if err != nil {
			return []models.WebhookResponse{}, fmt.Errorf("failed to parse %q webhook JSON: %w", event.Type, err)
		}
		return p.translateBalanceWebhook(ctx, accountReference, eventCreatedAt, balance)
	}

	return []models.WebhookResponse{}, fmt.Errorf("unhandled event type: %q", event.Type)
}

func (p *Plugin) translateBalanceWebhook(
	ctx context.Context,
	accountRef string,
	createdAt time.Time,
	balance stripe.Balance,
) ([]models.WebhookResponse, error) {
	responses := make([]models.WebhookResponse, 0, len(balance.Available))

	for _, a := range balance.Available {
		responses = append(responses, models.WebhookResponse{
			Balance: &models.PSPBalance{
				AccountReference: accountRef,
				Amount:           big.NewInt(a.Amount),
				Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, string(a.Currency)),
				CreatedAt:        createdAt,
			},
		})
	}
	return responses, nil
}
