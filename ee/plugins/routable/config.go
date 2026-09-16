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
// PayoutsPerMinute is per minute because that is the unit Routable negotiates
// and rate limits in: its RateLimit policies use a 60s window (§6.1.1) and the
// budget in §6.1 is written per minute. The engine's throttle is per second, so
// Plugin.PayoutsPerSecond does the conversion.
//
// Zero means unset - the shape of every config stored before this field
// existed - and the plugin answers it with DefaultPayoutsPerMinute. Nothing
// here defaults it, so the config stays a record of what was actually sent.
// The rate the plugin resolves is always > 0, which matters: the engine reads a
// 0 rate as "this connector has no payout task queue" and would route payouts
// back to the default queue, stranding the workflows already sitting on the
// dedicated one.
//
// Being unsigned is what rejects a negative rate, at unmarshal. Being whole
// rejects a fractional one there too: a per-minute budget has no need for
// fractions, and silently truncating 90.5 would be worse than refusing it.
type Config struct {
	APIKey           string `json:"apiKey" validate:"required"`
	Endpoint         string `json:"endpoint" validate:"omitempty,url"`
	ActingTeamMember string `json:"actingTeamMember"`
	PayoutsPerMinute uint64 `json:"payoutsPerMinute" validate:"omitempty,gt=0"`
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
	// omitempty is what lets 0 through to mean "unset": it skips the rules
	// after it on a zero value. No defaulting here - the plugin owns that.
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(cfg); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	return cfg, nil
}
