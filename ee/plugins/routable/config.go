package routable

import (
	"encoding/json"
	"strings"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

// PAGE_SIZE is the per-fetch page size advertised to the engine. Routable
// documents `page_size` as capped at 100 across v1 endpoints; we run at
// the maximum so a single FETCH_PAYMENTS cycle issues half the pagination
// requests we'd otherwise need at the default. At 200k tx/wk this is a
// meaningful Routable-side load reduction.
const PAGE_SIZE = 100

// Config is the connector configuration persisted by the engine and used to
// instantiate the Routable HTTP client.
//
// ActingTeamMember is optional at the connector level because Routable's
// POST /v1/payables call accepts the team member identifier on a per-request
// basis: callers may set the metadata key mappers.MetadataKeyActingTeamMember on
// the PSPPaymentInitiation to override (or supply) the team member without
// touching the connector config. If neither the config nor the metadata
// carry a value, the client returns a clear validation error before the
// request hits the wire.
type Config struct {
	APIKey           string                     `json:"apiKey" validate:"required"`
	Endpoint         string                     `json:"endpoint" validate:"omitempty,url"`
	ActingTeamMember string                     `json:"actingTeamMember"`
	PollingPeriod    sharedconfig.PollingPeriod `json:"pollingPeriod"`
}

// resolvedEndpoint returns the configured API endpoint, falling back to the
// public Routable URL when omitted. Sandbox is opt-in via explicit config.
func (c Config) resolvedEndpoint() string {
	if c.Endpoint == "" {
		return client.DefaultBaseURL
	}
	return c.Endpoint
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey           string `json:"apiKey"`
		Endpoint         string `json:"endpoint"`
		ActingTeamMember string `json:"actingTeamMember"`
		PollingPeriod    string `json:"pollingPeriod"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	pp, err := sharedconfig.NewPollingPeriod(raw.PollingPeriod, sharedconfig.DefaultPollingPeriod, sharedconfig.MinimumPollingPeriod)
	if err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	// Trim before validating so `"apiKey": "   "` doesn't slip past the
	// `required` rule and surface as an opaque 401 from the credential
	// probe at install-time.
	cfg := Config{
		APIKey:           strings.TrimSpace(raw.APIKey),
		Endpoint:         strings.TrimSpace(raw.Endpoint),
		ActingTeamMember: strings.TrimSpace(raw.ActingTeamMember),
		PollingPeriod:    pp,
	}
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(cfg); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	return cfg, nil
}
