package universal

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/mappers"
	"github.com/formancehq/payments/internal/models"
)

// Contract: counterparties advertising `features.webhookSignature ==
// "hmac-sha256"` MUST sign every delivery with these two headers. Signed
// payload is "<timestamp>.<body>"; timestamp is RFC3339 UTC.
const (
	WebhookHeaderSignature = "X-Universal-Signature"
	WebhookHeaderTimestamp = "X-Universal-Timestamp"

	// 5-minute skew tolerance matches Increase and the other Formance
	// connectors.
	signatureTolerance = 5 * time.Minute
)

// supportedWebhook is one entry in the install-time subscription list.
// Slice (vs map) keeps iteration order stable so subscription IDs in the
// counterparty stay deterministic across restarts.
type supportedWebhook struct {
	Name    string
	URLPath string
}

// supportedWebhooks is the canonical event catalogue. Order events and
// conversion events are intentionally absent: models.WebhookResponse has
// no Order/Conversion field — see contract/webhooks.md.
var supportedWebhooks = []supportedWebhook{
	{"account.created", "/account/created"},
	{"account.updated", "/account/updated"},
	{"external_account.created", "/external_account/created"},
	{"balance.updated", "/balance/updated"},
	{"payment.created", "/payment/created"},
	{"payment.updated", "/payment/updated"},
	{"payment.deleted", "/payment/deleted"},
	{"payment.cancelled", "/payment/cancelled"},
}

// supportedWebhookNames is the membership check used by TranslateWebhook
// (O(1) lookup against the canonical catalogue).
var supportedWebhookNames = func() map[string]struct{} {
	m := make(map[string]struct{}, len(supportedWebhooks))
	for _, w := range supportedWebhooks {
		m[w.Name] = struct{}{}
	}
	return m
}()

func (p *Plugin) CreateWebhooks(ctx context.Context, req models.CreateWebhooksRequest) (models.CreateWebhooksResponse, error) {
	declared, ok := p.declaredSet()
	if !ok {
		return models.CreateWebhooksResponse{}, plugins.ErrNotYetInstalled
	}
	if err := declared.require(models.CAPABILITY_CREATE_WEBHOOKS); err != nil {
		return models.CreateWebhooksResponse{}, err
	}
	if req.WebhookBaseUrl == "" {
		return models.CreateWebhooksResponse{}, errors.New("webhook base URL is empty")
	}
	if err := validateWebhookBaseURL(req.WebhookBaseUrl); err != nil {
		return models.CreateWebhooksResponse{}, err
	}

	configs := make([]models.PSPWebhookConfig, 0, len(supportedWebhooks))
	others := make([]models.PSPOther, 0, len(supportedWebhooks))

	for _, w := range supportedWebhooks {
		callback, err := url.JoinPath(req.WebhookBaseUrl, w.URLPath)
		if err != nil {
			return models.CreateWebhooksResponse{}, err
		}

		idemKey := fmt.Sprintf("%s:%s", req.ConnectorID, w.Name)
		resp, err := p.client.CreateWebhookSubscription(ctx, idemKey, &client.WebhookSubscriptionRequest{
			Name:        w.Name,
			CallbackURL: callback,
		})
		if err != nil {
			return models.CreateWebhooksResponse{}, fmt.Errorf("subscribing %s: %w", w.Name, err)
		}

		// PSPWebhookConfig never carries the secret — Formance stores
		// it server-side as part of the connector config and we read
		// it back via p.config.WebhookSharedSecret in VerifyWebhook.
		configs = append(configs, models.PSPWebhookConfig{Name: w.Name, URLPath: w.URLPath})
		raw, err := json.Marshal(resp)
		if err != nil {
			return models.CreateWebhooksResponse{}, err
		}
		others = append(others, models.PSPOther{ID: resp.ID, Other: raw})
	}

	p.logger.WithFields(map[string]any{
		"connector": p.name,
		"count":     len(configs),
	}).Info("registered universal webhook subscriptions")

	return models.CreateWebhooksResponse{Configs: configs, Others: others}, nil
}

// VerifyWebhook authenticates inbound deliveries. Runs in the engine's
// HTTP server process — Install never ran there, so we cannot consult
// declaredSet; gate on `client` (constructor-set, available in every
// process) and on the persisted WebhookSharedSecret. Install rejects an
// install where the counterparty advertised HMAC but no secret was
// provided, so a non-empty secret here is the install-time proof that
// signatures are expected.
func (p *Plugin) VerifyWebhook(_ context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	if p.client == nil {
		return models.VerifyWebhookResponse{}, plugins.ErrNotYetInstalled
	}

	signature := headerValue(req.Webhook.Headers, WebhookHeaderSignature)
	timestamp := headerValue(req.Webhook.Headers, WebhookHeaderTimestamp)
	secret := p.config.WebhookSharedSecret

	if secret == "" {
		// A signature header without a configured secret is spoofing or
		// drift — reject rather than silently accept.
		if signature != "" || timestamp != "" {
			return models.VerifyWebhookResponse{}, fmt.Errorf("%w: signature header present but no webhookSharedSecret configured", models.ErrWebhookVerification)
		}
		p.logger.WithField("connector", p.name).Debug("universal webhook accepted unsigned (no secret configured)")
		return models.VerifyWebhookResponse{}, nil
	}

	if signature == "" || timestamp == "" {
		return models.VerifyWebhookResponse{}, fmt.Errorf("%w: missing %s or %s header", models.ErrWebhookVerification, WebhookHeaderSignature, WebhookHeaderTimestamp)
	}

	ts, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("%w: invalid timestamp: %w", models.ErrWebhookVerification, err)
	}
	if delta := time.Since(ts); delta > signatureTolerance || delta < -signatureTolerance {
		return models.VerifyWebhookResponse{}, fmt.Errorf("%w: timestamp outside %s tolerance window", models.ErrWebhookVerification, signatureTolerance)
	}

	if !verifyHMACSHA256(secret, timestamp, req.Webhook.Body, signature) {
		return models.VerifyWebhookResponse{}, fmt.Errorf("%w: invalid signature", models.ErrWebhookVerification)
	}

	// Body MUST parse here. An unparseable body with a valid HMAC is a
	// contract violation and silently dropping the event-ID would lose
	// engine idempotency on retry.
	var ev client.WebhookEvent
	if err := json.Unmarshal(req.Webhook.Body, &ev); err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("%w: signed body is not a valid WebhookEvent (len=%d): %w", models.ErrWebhookVerification, len(req.Webhook.Body), err)
	}
	if ev.ID == "" {
		return models.VerifyWebhookResponse{}, fmt.Errorf("%w: signed body missing event id", models.ErrWebhookVerification)
	}
	return models.VerifyWebhookResponse{WebhookIdempotencyKey: &ev.ID}, nil
}

// TranslateWebhook runs in the engine's HTTP server process — see
// VerifyWebhook for why we gate on `client` only.
func (p *Plugin) TranslateWebhook(_ context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	if p.client == nil {
		return models.TranslateWebhookResponse{}, plugins.ErrNotYetInstalled
	}
	if _, ok := supportedWebhookNames[req.Name]; !ok {
		return models.TranslateWebhookResponse{}, fmt.Errorf("unknown webhook event %q", req.Name)
	}

	var ev client.WebhookEvent
	if err := json.Unmarshal(req.Webhook.Body, &ev); err != nil {
		return models.TranslateWebhookResponse{}, fmt.Errorf("decoding webhook body: %w", err)
	}

	resp, err := translateResource(req.Name, ev.Resource)
	if err != nil {
		return models.TranslateWebhookResponse{}, err
	}
	return models.TranslateWebhookResponse{Responses: []models.WebhookResponse{resp}}, nil
}

// translateResource dispatches by event name to the matching resource on
// WebhookResponse. Missing resource on the wire fails loudly so a
// malformed delivery doesn't silently produce a no-op.
func translateResource(name string, r client.WebhookResource) (models.WebhookResponse, error) {
	switch name {
	case "account.created", "account.updated":
		if r.Account == nil {
			return models.WebhookResponse{}, fmt.Errorf("%s requires resource.account", name)
		}
		acc, err := mappers.AccountToPSPAccount(*r.Account)
		if err != nil {
			return models.WebhookResponse{}, err
		}
		return models.WebhookResponse{Account: &acc}, nil
	case "external_account.created":
		if r.ExternalAccount == nil {
			return models.WebhookResponse{}, fmt.Errorf("%s requires resource.externalAccount", name)
		}
		acc, err := mappers.AccountToPSPAccount(*r.ExternalAccount)
		if err != nil {
			return models.WebhookResponse{}, err
		}
		return models.WebhookResponse{ExternalAccount: &acc}, nil
	case "balance.updated":
		if r.Balance == nil {
			return models.WebhookResponse{}, fmt.Errorf("%s requires resource.balance", name)
		}
		b, err := mappers.BalanceToPSPBalance(*r.Balance)
		if err != nil {
			return models.WebhookResponse{}, err
		}
		return models.WebhookResponse{Balance: &b}, nil
	case "payment.created", "payment.updated":
		if r.Payment == nil {
			return models.WebhookResponse{}, fmt.Errorf("%s requires resource.payment", name)
		}
		pay, err := mappers.PaymentToPSPPayment(*r.Payment)
		if err != nil {
			return models.WebhookResponse{}, err
		}
		return models.WebhookResponse{Payment: &pay}, nil
	case "payment.deleted":
		if r.PaymentToDelete == nil {
			return models.WebhookResponse{}, fmt.Errorf("%s requires resource.paymentToDelete", name)
		}
		return models.WebhookResponse{PaymentToDelete: &models.PSPPaymentsToDelete{Reference: *r.PaymentToDelete}}, nil
	case "payment.cancelled":
		if r.PaymentToCancel == nil {
			return models.WebhookResponse{}, fmt.Errorf("%s requires resource.paymentToCancel", name)
		}
		return models.WebhookResponse{PaymentToCancel: &models.PSPPaymentsToCancel{Reference: *r.PaymentToCancel}}, nil
	default:
		return models.WebhookResponse{}, fmt.Errorf("unsupported webhook event %q", name)
	}
}

// headerValue is a case-insensitive header lookup: prefer the direct hit,
// fall back to a scan if the transport hasn't canonicalised keys yet.
func headerValue(headers map[string][]string, key string) string {
	if vs := headers[key]; len(vs) > 0 {
		return vs[0]
	}
	for k, vs := range headers {
		if strings.EqualFold(k, key) && len(vs) > 0 {
			return vs[0]
		}
	}
	return ""
}

// validateWebhookBaseURL enforces HTTPS unless the hostname is
// unambiguously local: `localhost`, loopback IPs, bare docker-service
// names (no dot), `.local`, `.localhost`. Anything else MUST be HTTPS.
func validateWebhookBaseURL(raw string) error {
	if strings.HasPrefix(raw, "https://") {
		return nil
	}
	if !strings.HasPrefix(raw, "http://") {
		return errors.New("webhook base URL must use https:// (or http:// for local hostnames only)")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid webhook base URL %q: %w", raw, err)
	}
	host := u.Hostname()
	if isLocalHostname(host) {
		return nil
	}
	return fmt.Errorf("webhook base URL must use HTTPS (got http://%s)", host)
}

func isLocalHostname(host string) bool {
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	if !strings.Contains(host, ".") {
		return true
	}
	return strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".localhost")
}

func verifyHMACSHA256(secret, timestamp string, body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := mac.Sum(nil)

	got, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(expected, got) == 1
}
