package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
	stripesdk "github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/webhook"
	"math/big"
	"net/url"
	"strings"
	"time"
)

const (
	webhookIDMetadataKey         = "webhook_id"
	webhookSecretMetadataKey     = "secret"
	webhookStripeSignatureHeader = "Stripe-Signature"
)

type supportedWebhook struct {
	urlPath string
	fn      func(context.Context, *stripesdk.Event) (models.TranslateWebhookResponse, error)
}

func (p *Plugin) initWebhookConfig() {
	p.supportedWebhooks = map[stripesdk.EventType]supportedWebhook{
		// Balances
		stripesdk.EventTypeBalanceAvailable: {
			urlPath: "/balance-available",
			fn:      p.translateBalanceAvailable,
		},

		// Payouts
		stripesdk.EventTypePayoutCreated: {
			urlPath: "/payout/created",
			fn:      p.translatePayoutEvents,
		},
		stripesdk.EventTypePayoutUpdated: {
			urlPath: "/payout/updated",
			fn:      p.translatePayoutEvents,
		},
		stripesdk.EventTypePayoutFailed: {
			urlPath: "/payout/failed",
			fn:      p.translatePayoutEvents,
		},
		stripesdk.EventTypePayoutPaid: {
			urlPath: "/payout/paid",
			fn:      p.translatePayoutEvents,
		},
		stripesdk.EventTypePayoutCanceled: {
			urlPath: "/payout/canceled",
			fn:      p.translatePayoutEvents,
		},

		// NOTE: No events: for cancelling, refund or failure...?
		// It is implied from balance transaction type in the current implementation
		// Transfers
		stripesdk.EventTypeTransferCreated: {
			urlPath: "/transfer/created",
			fn:      p.translateTransferEvents,
		},
		stripesdk.EventTypeTransferUpdated: {
			urlPath: "/transfer/updated",
			fn:      p.translateTransferEvents,
		},
		stripesdk.EventTypeTransferReversed: {
			urlPath: "/transfer/reversed",
			fn:      p.translateTransferEvents,
		},

		// Accounts
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

		resp, err := p.client.CreateWebhook(ctx, webhookURL, req.ConnectorID, eventType)
		if err != nil {
			return models.CreateWebhooksResponse{}, err
		}

		configs = append(configs, models.PSPWebhookConfig{
			Name:    string(eventType),
			URLPath: w.urlPath,
			Metadata: map[string]string{
				webhookSecretMetadataKey: resp.Secret,
				webhookIDMetadataKey:     resp.WebhookEndpointID,
			},
		})
	}

	return models.CreateWebhooksResponse{
		Configs: configs,
	}, nil
}

func (p *Plugin) verifyWebhook(_ context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	signatureHeader, err := getStripeSignatureFromHeader(req.Webhook.Headers)
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("failed to get signature header: %w", err)
	}

	secret := req.Config.Metadata[webhookSecretMetadataKey]

	// You can just validate with the following function, but we need to get an idempotency key.
	// err := webhook.ValidatePayload(req.Webhook.Body, signatureHeader, secret)
	event, err := webhook.ConstructEvent(req.Webhook.Body, signatureHeader, secret)

	if err != nil {
		return models.VerifyWebhookResponse{}, err
	}

	return models.VerifyWebhookResponse{
		WebhookIdempotencyKey: &event.ID,
	}, nil
}

func (p *Plugin) translateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	config := req.Config
	if config == nil || config.Metadata == nil || config.Metadata[webhookSecretMetadataKey] == "" {
		return models.TranslateWebhookResponse{}, fmt.Errorf("missing config properties")
	}

	secret := config.Metadata[webhookSecretMetadataKey]

	signatureHeader, err := getStripeSignatureFromHeader(req.Webhook.Headers)
	if err != nil {
		return models.TranslateWebhookResponse{}, fmt.Errorf("failed to get signature header: %w", err)
	}
	event, err := webhook.ConstructEvent(req.Webhook.Body, signatureHeader, secret)
	if err != nil {
		return models.TranslateWebhookResponse{}, fmt.Errorf("failed to construct event: %w", err)
	}

	return p.translateBalanceAvailable(ctx, &event)
}

func (p *Plugin) translateBalanceAvailable(ctx context.Context, event *stripesdk.Event) (models.TranslateWebhookResponse, error) {
	var balanceAvailable stripesdk.Balance

	err := json.Unmarshal(event.Data.Raw, &balanceAvailable)
	if err != nil {
		return models.TranslateWebhookResponse{}, err
	}

	accountReference := getAccountReference(event)

	pspBalances := p.fromStripeBalanceToPSPBalances(&balanceAvailable, accountReference)

	balancesResp := make([]models.WebhookResponse, 0, len(pspBalances))
	for _, balance := range pspBalances {
		balancesResp = append(balancesResp, models.WebhookResponse{
			Balance: &balance,
		})
	}

	return models.TranslateWebhookResponse{
		Responses: balancesResp,
	}, nil
}

func (p *Plugin) translatePayoutEvents(ctx context.Context, event *stripesdk.Event) (models.TranslateWebhookResponse, error) {
	var payout stripesdk.Payout
	err := json.Unmarshal(event.Data.Raw, &payout)
	if err != nil {
		return models.TranslateWebhookResponse{}, err
	}

	accountReference := getAccountReference(event)

	rawData, err := json.Marshal(event)
	if err != nil {
		return models.TranslateWebhookResponse{}, fmt.Errorf("failed to marshal raw data: %w", err)
	}
	metadata := make(map[string]string)

	payment, err := mapFromPayoutToPSPPayment(
		&payout,
		payout.BalanceTransaction.ID,
		&accountReference,
		nil,
		metadata,
		rawData,
	)

	if err != nil {
		return models.TranslateWebhookResponse{}, fmt.Errorf("failed to map payout to PSP payment: %w", err)
	}

	return models.TranslateWebhookResponse{
		Responses: []models.WebhookResponse{
			{
				Payment: payment,
			},
		},
	}, nil
}

func (p *Plugin) translateTransferEvents(ctx context.Context, event *stripesdk.Event) (models.TranslateWebhookResponse, error) {
	var transfer stripesdk.Transfer
	err := json.Unmarshal(event.Data.Raw, &transfer)
	if err != nil {
		return models.TranslateWebhookResponse{}, err
	}

	transactionCurrency := strings.ToUpper(string(transfer.Currency))
	_, ok := supportedCurrenciesWithDecimal[transactionCurrency]
	if !ok {
		return models.TranslateWebhookResponse{}, fmt.Errorf("%w %q", ErrUnsupportedCurrency, transactionCurrency)
	}

	accountReference := getAccountReference(event)
	metadata := make(map[string]string)
	appendMetadata(metadata, transfer.Metadata)

	status := models.PAYMENT_STATUS_SUCCEEDED
	switch event.Type {
	case stripesdk.EventTypeTransferCreated:
		status = models.PAYMENT_STATUS_PENDING
	// Occurs whenever a transfer's description or metadata is updated.
	case stripesdk.EventTypeTransferUpdated:

	// Occurs whenever a transfer is reversed, including partial reversals.
	case stripesdk.EventTypeTransferReversed:

	}

	payment := &models.PSPPayment{
		Reference:              transfer.BalanceTransaction.ID,
		Type:                   models.PAYMENT_TYPE_TRANSFER,
		Status:                 status,
		Amount:                 big.NewInt(transfer.Amount - transfer.AmountReversed),
		Asset:                  currency.FormatAsset(supportedCurrenciesWithDecimal, transactionCurrency),
		Scheme:                 models.PAYMENT_SCHEME_OTHER,
		CreatedAt:              time.Unix(transfer.Created, 0),
		SourceAccountReference: &accountReference,
		Raw:                    event.Data.Raw,
		Metadata:               metadata,
	}

	if transfer.Destination != nil {
		payment.DestinationAccountReference = &transfer.Destination.ID
	}

	return models.TranslateWebhookResponse{}, nil
}

func getAccountReference(event *stripesdk.Event) string {
	if event.Account == "" {
		return rootAccountReference
	}
	return event.Account
}

func getStripeSignatureFromHeader(headers map[string][]string) (string, error) {
	signatureHeader := headers[webhookStripeSignatureHeader]
	if len(signatureHeader) == 0 {
		return "", fmt.Errorf("missing signature header")
	}
	return signatureHeader[0], nil
}
