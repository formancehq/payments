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

// PAGE_SIZE matches Routable's documented page_size cap across v1
// endpoints (100). See MAPPINGS.md §6.1 for the throughput math.
const PAGE_SIZE = 100

// Config is documented in MAPPINGS.md §1. ActingTeamMember is
// connector-level optional because callers can override it per-request
// via the MetadataKeyActingTeamMember key on the PSPPaymentInitiation.
type Config struct {
	APIKey           string                     `json:"apiKey" validate:"required"`
	Endpoint         string                     `json:"endpoint" validate:"omitempty,url"`
	ActingTeamMember string                     `json:"actingTeamMember"`
	PollingPeriod    sharedconfig.PollingPeriod `json:"pollingPeriod"`
}

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

	// Trim so `"apiKey": "   "` doesn't slip past the required rule.
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
