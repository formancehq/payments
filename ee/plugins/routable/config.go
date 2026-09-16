package routable

import (
	"encoding/json"
	"strings"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

// PAGE_SIZE matches Routable's documented page_size cap across v1
// endpoints (100). See MAPPINGS.md §6.1 for the throughput math.
const PAGE_SIZE = 100

// Config is documented in MAPPINGS.md §1. ActingTeamMember is
// connector-level optional because callers can override it per-request
// via the MetadataKeyActingTeamMember key on the PSPPaymentInitiation.
//
// PayoutsPerMinute is per minute because that is the unit Routable rate limits
// in (MAPPINGS.md §6.1.1). It is left as stored - 0 means unset - so the config
// stays a record of what was actually sent; resolvedPayoutsPerMinute applies
// the default. The unsigned, whole type is what rejects a negative or
// fractional rate, at unmarshal.
type Config struct {
	APIKey           string `json:"apiKey" validate:"required"`
	Endpoint         string `json:"endpoint" validate:"omitempty,url"`
	ActingTeamMember string `json:"actingTeamMember"`
	PayoutsPerMinute uint64 `json:"payoutsPerMinute"`
}

func (c Config) resolvedEndpoint() string {
	if c.Endpoint == "" {
		return client.DefaultBaseURL
	}
	return c.Endpoint
}

// resolvedPayoutsPerMinute applies DefaultPayoutsPerMinute when the config
// leaves the rate unset, which is how every config written before the field
// existed reads. The result is always > 0: the engine reads a 0 rate as "this
// connector has no dedicated payout queue".
func (c Config) resolvedPayoutsPerMinute() uint64 {
	if c.PayoutsPerMinute == 0 {
		return DefaultPayoutsPerMinute
	}
	return c.PayoutsPerMinute
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey           string `json:"apiKey"`
		Endpoint         string `json:"endpoint"`
		ActingTeamMember string `json:"actingTeamMember"`
		PayoutsPerMinute uint64 `json:"payoutsPerMinute"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	// Trim so `"apiKey": "   "` doesn't slip past the required rule.
	cfg := Config{
		APIKey:           strings.TrimSpace(raw.APIKey),
		Endpoint:         strings.TrimSpace(raw.Endpoint),
		ActingTeamMember: strings.TrimSpace(raw.ActingTeamMember),
		PayoutsPerMinute: raw.PayoutsPerMinute,
	}
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(cfg); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	return cfg, nil
}
