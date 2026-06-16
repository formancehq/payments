package krakenpro

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

// Config is the Kraken Pro connector config (see MAPPINGS §2). Endpoint
// is required, not optional: this client speaks the Pro VIP dialect (JSON
// body, lowercase headers) which is incompatible with the public Spot API
// (form-encoded, API-Key header), so a blank endpoint must fail fast
// rather than silently fall back to the public host.
type Config struct {
	APIKey        string                     `json:"apiKey" validate:"required"`
	APISecret     string                     `json:"apiSecret" validate:"required"`
	Endpoint      string                     `json:"endpoint" validate:"required,url"`
	PollingPeriod sharedconfig.PollingPeriod `json:"pollingPeriod"`
}

// PAGE_SIZE is the per-call page bound for Ledgers / ClosedOrders.
// Kraken documents no hard cap; 50 is the observed default and also the
// short-page signal the frozen-window walk uses to detect drain.
const PAGE_SIZE = 50

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey        string `json:"apiKey"`
		APISecret     string `json:"apiSecret"`
		Endpoint      string `json:"endpoint"`
		PollingPeriod string `json:"pollingPeriod"`
	}

	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	pp, err := sharedconfig.NewPollingPeriod(
		raw.PollingPeriod,
		sharedconfig.GetDefaultPollingPeriod(),
		sharedconfig.GetMinimumPollingPeriod(),
	)
	if err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	config := Config{
		APIKey:        raw.APIKey,
		APISecret:     raw.APISecret,
		Endpoint:      raw.Endpoint,
		PollingPeriod: pp,
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	return config, nil
}
