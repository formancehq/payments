package bitstamp

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

// Config is the Bitstamp connector config. See MAPPINGS §2 for the
// rationale on the deliberately minimal surface (no accountScope,
// derivatives, or per-source toggles — the PSP is the source of truth).
type Config struct {
	APIKey    string `json:"apiKey" validate:"required"`
	APISecret string `json:"apiSecret" validate:"required"`
	Endpoint  string `json:"endpoint" validate:"omitempty,url"`
}

const PAGE_SIZE = 100

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey    string `json:"apiKey"`
		APISecret string `json:"apiSecret"`
		Endpoint  string `json:"endpoint"`
	}

	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	config := Config{
		APIKey:    raw.APIKey,
		APISecret: raw.APISecret,
		Endpoint:  raw.Endpoint,
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	return config, nil
}
