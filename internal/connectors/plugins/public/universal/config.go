package universal

import (
	"encoding/json"
	"strings"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

const PAGE_SIZE = 100

// Config is the install-time configuration of the Universal Connector.
//
// Endpoint must point at a counterparty server implementing
// contract/universal-openapi.yaml (v1). APIKey is sent as a Bearer token.
//
// WebhookSharedSecret is required when the counterparty's
// /v1/capabilities response sets features.webhookSignature to "hmac-sha256"
// OR features.eventStream to "wss" — both transports share the same HMAC
// scheme, so one secret covers both. The plugin never logs nor surfaces
// these secrets.
//
// StreamEndpoint is an optional override for the WebSocket dial target.
// When empty and the counterparty advertises features.eventStream == "wss",
// the supervisor derives it from Endpoint by swapping http→ws / https→wss.
//
// CapabilityOverrides is an optional allow-list to *narrow* what the
// counterparty advertised: any capability listed here that the counterparty
// did not declare is rejected at install. Use it to disable a primitive on
// a per-install basis without touching the counterparty.
//
// Encoded as a comma-separated string ("FETCH_ACCOUNTS,FETCH_PAYMENTS")
// because the connector registry's reflection-driven OpenAPI schema only
// supports scalar types. Empty means "use everything the counterparty
// advertised".
type Config struct {
	Endpoint            string                     `json:"endpoint" validate:"required,url"`
	APIKey              string                     `json:"apiKey" validate:"required"`
	WebhookSharedSecret string                     `json:"webhookSharedSecret" validate:"omitempty"`
	StreamEndpoint      string                     `json:"streamEndpoint" validate:"omitempty,url"`
	PollingPeriod       sharedconfig.PollingPeriod `json:"pollingPeriod"`
	CapabilityOverrides string                     `json:"capabilityOverrides" validate:"omitempty"`
}

// CapabilityOverridesList parses the comma-separated CapabilityOverrides
// into a clean string slice (trimmed, empty entries dropped). Validated
// against the canonical capability names by parseDeclaredCapabilities.
func (c Config) CapabilityOverridesList() []string {
	if c.CapabilityOverrides == "" {
		return nil
	}
	parts := strings.Split(c.CapabilityOverrides, ",")
	out := parts[:0]
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		Endpoint            string `json:"endpoint"`
		APIKey              string `json:"apiKey"`
		WebhookSharedSecret string `json:"webhookSharedSecret"`
		StreamEndpoint      string `json:"streamEndpoint"`
		PollingPeriod       string `json:"pollingPeriod"`
		CapabilityOverrides string `json:"capabilityOverrides"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	pp, err := sharedconfig.NewPollingPeriod(
		raw.PollingPeriod,
		sharedconfig.DefaultPollingPeriod,
		sharedconfig.MinimumPollingPeriod,
	)
	if err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	cfg := Config{
		Endpoint:            raw.Endpoint,
		APIKey:              raw.APIKey,
		WebhookSharedSecret: raw.WebhookSharedSecret,
		StreamEndpoint:      raw.StreamEndpoint,
		PollingPeriod:       pp,
		CapabilityOverrides: raw.CapabilityOverrides,
	}
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(cfg); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	for _, c := range cfg.CapabilityOverridesList() {
		var cap models.Capability
		if err := cap.Scan(c); err != nil {
			return Config{}, errors.Wrap(models.ErrInvalidConfig, "unknown capability override "+c)
		}
	}
	return cfg, nil
}
