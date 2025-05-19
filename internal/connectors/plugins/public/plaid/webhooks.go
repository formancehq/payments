package plaid

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/plaid/plaid-go/v34/plaid"
)

type supportedWebhook struct {
	urlPath string
	fn      func(context.Context, models.TranslateWebhookRequest) ([]models.WebhookResponse, error)
}

func (p *Plugin) initWebhookConfig() {
	p.supportedWebhooks = map[string]supportedWebhook{
		"all": {
			urlPath: "/all",
			fn:      p.handleAllWebhook,
		},
	}
}

func (p *Plugin) createWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	configs := make([]models.PSPWebhookConfig, 0, len(p.supportedWebhooks))
	for name, w := range p.supportedWebhooks {
		configs = append(configs, models.PSPWebhookConfig{
			Name:    name,
			URLPath: w.urlPath,
		})
	}

	return models.CreateWebhooksResponse{
		Configs: configs,
	}, nil
}

func (p *Plugin) verifyWebhook(ctx context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	// Extract the signed JWT from the webhook header
	tokenStrings := req.Webhook.Headers["plaid-verification"]
	if len(tokenStrings) != 1 {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid token: %w", models.ErrInvalidRequest)
	}

	// Verify the token using the public key
	token, err := jwt.Parse(tokenStrings[0], func(token *jwt.Token) (interface{}, error) {
		// Extract the key ID (kid) from the token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing kid in token header: %w", models.ErrInvalidRequest)
		}

		key, err := p.client.GetWebhookVerificationKey(ctx, kid)
		if err != nil {
			return nil, err
		}

		if key.ExpiredAt.IsSet() {
			// reject expired keys
			return nil, fmt.Errorf("expired key: %w", models.ErrInvalidRequest)
		}

		// Signing key must be an ecdsa.PublicKey struct
		publicKey := new(ecdsa.PublicKey)
		publicKey.Curve = elliptic.P256()
		x, _ := base64.URLEncoding.DecodeString(key.X + "=")
		xc := new(big.Int)
		publicKey.X = xc.SetBytes(x)
		y, _ := base64.URLEncoding.DecodeString(key.Y + "=")
		yc := new(big.Int)
		publicKey.Y = yc.SetBytes(y)

		return publicKey, nil
	}, jwt.WithValidMethods([]string{"ES256"}))
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("failed to verify token: %w", err)
	}

	// Check if the token is valid
	if !token.Valid {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid token: %w", models.ErrInvalidRequest)
	}

	return models.VerifyWebhookResponse{}, nil
}

func (p *Plugin) translateWebhook(ctx context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	webhookTranslator, ok := p.supportedWebhooks[req.Name]
	if !ok {
		return models.TranslateWebhookResponse{}, fmt.Errorf("unsupported webhook event type: %s", req.Name)
	}

	resp, err := webhookTranslator.fn(ctx, req)
	if err != nil {
		return models.TranslateWebhookResponse{}, err
	}

	return models.TranslateWebhookResponse{
		Responses: resp,
	}, nil
}

func (p *Plugin) handleAllWebhook(ctx context.Context, req models.TranslateWebhookRequest) ([]models.WebhookResponse, error) {
	baseWebhook, err := p.client.BaseWebhookTranslation(req.Webhook.Body)
	if err != nil {
		return nil, err
	}

	switch baseWebhook.WebhookType {
	case plaid.WEBHOOKTYPE_ASSETS:
		return p.handleAssetsWebhook(ctx, req, baseWebhook)
	case plaid.WEBHOOKTYPE_AUTH:
		return p.handleAuthWebhook(ctx, req, baseWebhook)
	case plaid.WEBHOOKTYPE_HOLDINGS:
		return p.handleHoldingsWebhook(ctx, req, baseWebhook)
	case plaid.WEBHOOKTYPE_INVESTMENTS_TRANSACTIONS:
		return p.handleInvestmentsTransactionsWebhook(ctx, req, baseWebhook)
	case plaid.WEBHOOKTYPE_ITEM:
		return p.handleItemWebhook(ctx, req, baseWebhook)
	case plaid.WEBHOOKTYPE_LIABILITIES:
		return p.handleLiabilitiesWebhook(ctx, req, baseWebhook)
	case plaid.WEBHOOKTYPE_TRANSACTIONS:
		return p.handleTransactionsWebhook(ctx, req, baseWebhook)
	default:
		return []models.WebhookResponse{}, fmt.Errorf("unsupported webhook type: %s", baseWebhook.WebhookType)
	}
}

func (p *Plugin) handleAssetsWebhook(context.Context, models.TranslateWebhookRequest, client.BaseWebhooks) ([]models.WebhookResponse, error) {
	// Not interested in assets webhooks
	return []models.WebhookResponse{}, nil
}

func (p *Plugin) handleAuthWebhook(ctx context.Context, req models.TranslateWebhookRequest, baseWebhook client.BaseWebhooks) ([]models.WebhookResponse, error) {
	// Not interested in auth webhooks
	return []models.WebhookResponse{}, nil
}

func (p *Plugin) handleHoldingsWebhook(ctx context.Context, req models.TranslateWebhookRequest, baseWebhook client.BaseWebhooks) ([]models.WebhookResponse, error) {
	// Not interested in holdings webhooks
	return []models.WebhookResponse{}, nil
}

func (p *Plugin) handleInvestmentsTransactionsWebhook(ctx context.Context, req models.TranslateWebhookRequest, baseWebhook client.BaseWebhooks) ([]models.WebhookResponse, error) {
	// Not interested in investments transactions webhooks
	return []models.WebhookResponse{}, nil
}

func (p *Plugin) handleItemWebhook(ctx context.Context, req models.TranslateWebhookRequest, baseWebhook client.BaseWebhooks) ([]models.WebhookResponse, error) {
	switch baseWebhook.WebhookCode {
	case "ERROR":
	case "LOGIN_REPAIRED":
	case "NEW_ACCOUNTS_AVAILABLE":
	case "PENDING_DISCONNECT":
	case "PENDING_EXPIRATION":
	case "USER_PERMISSION_REVOKED":
	case "USER_ACCOUNT_REVOKED":
	case "WEBHOOK_UPDATE_ACKNOWLEDGED":
	}

	return []models.WebhookResponse{}, nil
}

func (p *Plugin) handleLiabilitiesWebhook(ctx context.Context, req models.TranslateWebhookRequest, baseWebhook client.BaseWebhooks) ([]models.WebhookResponse, error) {
	// Not interested in liabilities webhooks
	return []models.WebhookResponse{}, nil
}

func (p *Plugin) handleTransactionsWebhook(ctx context.Context, req models.TranslateWebhookRequest, baseWebhook client.BaseWebhooks) ([]models.WebhookResponse, error) {
	switch baseWebhook.WebhookCode {
	case "SYNC_UPDATES_AVAILABLE":
	case "RECURRING_TRANSACTIONS_UPDATE":
	case "INITIAL_UPDATE":
	case "HISTORICAL_UPDATE":
	case "DEFAULT_UPDATE":
	case "TRANSACTIONS_REMOVED":
	}
	return []models.WebhookResponse{}, nil
}
