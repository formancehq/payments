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

// Header names the contract reserves for delivered webhook signatures. Both
// headers are required when the counterparty advertised
// features.webhookSignature == "hmac-sha256". Timestamp is RFC3339 UTC; the
// signed payload is "<timestamp>.<body>".
const (
	WebhookHeaderSignature = "X-Universal-Signature"
	WebhookHeaderTimestamp = "X-Universal-Timestamp"

	// signatureTolerance bounds clock skew between Formance and the
	// counterparty when verifying delivered webhooks. 5 minutes matches
	// what every other Formance connector uses (see Increase).
	signatureTolerance = 5 * time.Minute
)

// supportedWebhooks lists every event the contract subscribes to at
// install. Each entry maps the canonical event name → the URL suffix
// that the engine will route inbound deliveries on
// (see internal/api/v3/router.go and engine.HandleWebhook).
//
// Order and conversion events are deliberately absent: the engine's
// models.WebhookResponse struct exposes no Order/Conversion fields, so
// TranslateWebhook has no way to surface those resources to the engine.
// Orders and conversions are kept on the periodic `FetchNextOrders` /
// `FetchNextConversions` poll — see contract/state-machines.md for the
// rationale. Counterparties that want lower latency for those should
// shorten the connector's pollingPeriod (subject to the 20-minute floor).
var supportedWebhooks = map[string]string{
	"account.created":          "/account/created",
	"account.updated":          "/account/updated",
	"external_account.created": "/external_account/created",
	"balance.updated":          "/balance/updated",
	"payment.created":          "/payment/created",
	"payment.updated":          "/payment/updated",
	"payment.deleted":          "/payment/deleted",
	"payment.cancelled":        "/payment/cancelled",
}

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

	for eventType, urlPath := range supportedWebhooks {
		callback, err := url.JoinPath(req.WebhookBaseUrl, urlPath)
		if err != nil {
			return models.CreateWebhooksResponse{}, err
		}

		idemKey := fmt.Sprintf("%s:%s", req.ConnectorID, eventType)
		resp, err := p.client.CreateWebhookSubscription(ctx, idemKey, &client.WebhookSubscriptionRequest{
			Name:        eventType,
			CallbackURL: callback,
		})
		if err != nil {
			return models.CreateWebhooksResponse{}, fmt.Errorf("subscribing %s: %w", eventType, err)
		}

		configs = append(configs, models.PSPWebhookConfig{
			Name:    eventType,
			URLPath: urlPath,
			// We never store the secret here — Formance keeps it on the
			// connector config server-side and we read it back via
			// p.config.WebhookSharedSecret in VerifyWebhook.
		})
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

func (p *Plugin) VerifyWebhook(_ context.Context, req models.VerifyWebhookRequest) (models.VerifyWebhookResponse, error) {
	// Only check `client` (set in New, available in every process) — we
	// can NOT consult declaredSet here because VerifyWebhook is
	// dispatched in the engine's HTTP server process, while Install runs
	// in the worker process. The two processes hold separate plugin
	// instances, so `declared` is always nil here. The engine would not
	// have routed this webhook to us at all unless a WebhookConfig was
	// previously stored at install — its presence is the implicit
	// capability proof.
	if p.client == nil {
		return models.VerifyWebhookResponse{}, plugins.ErrNotYetInstalled
	}

	// HMAC verification when the user provided a shared secret. The
	// `features.WebhookSignature` value from /v1/capabilities only
	// reaches the worker process — in the server process we infer
	// "verify if a secret was provided, skip otherwise". Production
	// installs MUST set the secret; absence yields a logged warning so
	// misconfiguration is visible.
	secret := p.config.WebhookSharedSecret
	if secret == "" {
		p.logger.WithField("connector", p.name).Info("universal webhook signature verification skipped (no webhookSharedSecret configured)")
		return models.VerifyWebhookResponse{WebhookIdempotencyKey: nil}, nil
	}

	signature := headerValue(req.Webhook.Headers, WebhookHeaderSignature)
	timestamp := headerValue(req.Webhook.Headers, WebhookHeaderTimestamp)
	if signature == "" || timestamp == "" {
		return models.VerifyWebhookResponse{}, errors.New("missing webhook signature or timestamp header")
	}

	ts, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return models.VerifyWebhookResponse{}, fmt.Errorf("invalid timestamp: %w", err)
	}
	if delta := time.Since(ts); delta > signatureTolerance || delta < -signatureTolerance {
		return models.VerifyWebhookResponse{}, errors.New("timestamp outside tolerance window")
	}

	if !verifyHMACSHA256(secret, timestamp, req.Webhook.Body, signature) {
		return models.VerifyWebhookResponse{}, errors.New("invalid webhook signature")
	}

	// Idempotency-key surface to the engine: every event ID dedups across
	// retries. If the body fails to parse we let TranslateWebhook surface
	// the error — VerifyWebhook only attests authenticity.
	var ev client.WebhookEvent
	if err := json.Unmarshal(req.Webhook.Body, &ev); err == nil && ev.ID != "" {
		return models.VerifyWebhookResponse{WebhookIdempotencyKey: &ev.ID}, nil
	}
	return models.VerifyWebhookResponse{}, nil
}

func (p *Plugin) TranslateWebhook(_ context.Context, req models.TranslateWebhookRequest) (models.TranslateWebhookResponse, error) {
	// As with VerifyWebhook above: TranslateWebhook runs in the engine's
	// HTTP server process where Install never ran. Gate on `client` only
	// (constructor-set, available everywhere); the engine wouldn't have
	// dispatched the event at all without a stored WebhookConfig.
	if p.client == nil {
		return models.TranslateWebhookResponse{}, plugins.ErrNotYetInstalled
	}
	if _, ok := supportedWebhooks[req.Name]; !ok {
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

// translateResource maps a parsed WebhookEvent to the engine's WebhookResponse
// shape. The dispatch is by event name so the counterparty can use whichever
// resource subset matches that event; we validate the expected resource is
// present so a malformed payload fails loudly.
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

func headerValue(headers map[string][]string, key string) string {
	if vs := headers[key]; len(vs) > 0 {
		return vs[0]
	}
	// HTTP headers are case-insensitive; fall back to a slow scan when the
	// transport already canonicalised them.
	for k, vs := range headers {
		if strings.EqualFold(k, key) && len(vs) > 0 {
			return vs[0]
		}
	}
	return ""
}

// validateWebhookBaseURL enforces HTTPS on any base URL the engine
// hands us, unless the hostname is unambiguously local. This mirrors
// what most real PSPs do for development tunnels: HTTPS in production,
// HTTP allowed only for `localhost`, loopback IPs, or unqualified
// hostnames (e.g. docker-compose service names like `payments`,
// `payments:8080`). Anything with a dot in the host that isn't `.local`
// or `.localhost` MUST be HTTPS.
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
	// Bare hostnames (no dot) are docker-internal service names —
	// not reachable from the public internet, safe to ship over HTTP.
	if !strings.Contains(host, ".") {
		return true
	}
	// Conventional local TLDs.
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
