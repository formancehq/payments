package bitstamp

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

// Config is the Bitstamp connector config. See MAPPINGS §2 for the
// rationale on the deliberately minimal surface (no accountScope,
// derivatives, or per-source toggles — the PSP is the source of truth).
type Config struct {
	APIKey        string                     `json:"apiKey" validate:"required"`
	APISecret     string                     `json:"apiSecret" validate:"required"`
	Endpoint      string                     `json:"endpoint" validate:"omitempty,url"`
	PollingPeriod sharedconfig.PollingPeriod `json:"pollingPeriod"`
}

const PAGE_SIZE = 100

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
