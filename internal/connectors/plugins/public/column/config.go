package column

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	APIKey   string `json:"apiKey" validate:"required"`
	Endpoint string `json:"endpoint" validate:"required,url"`
}

const PAGE_SIZE = 100 // mapx page size is 100

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey   string `json:"apiKey"`
		Endpoint string `json:"endpoint"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	config := Config{
		APIKey:   raw.APIKey,
		Endpoint: raw.Endpoint,
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
