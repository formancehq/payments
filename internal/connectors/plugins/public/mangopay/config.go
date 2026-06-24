package mangopay

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	ClientID string `json:"clientID" validate:"required"`
	APIKey   string `json:"apiKey" validate:"required"`
	Endpoint string `json:"endpoint" validate:"required"`
}

const PAGE_SIZE = 100 // max page size is 100

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		ClientID string `json:"clientID"`
		APIKey   string `json:"apiKey"`
		Endpoint string `json:"endpoint"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	config := Config{
		ClientID: raw.ClientID,
		APIKey:   raw.APIKey,
		Endpoint: raw.Endpoint,
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
