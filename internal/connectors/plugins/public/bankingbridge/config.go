package bankingbridge

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
)

type Config struct {
	ClientID     string `json:"clientID" validate:"required"`
	ClientSecret string `json:"clientSecret" validate:"required"`
	Endpoint     string `json:"endpoint" validate:"required,uri"`
	AuthEndpoint string `json:"authEndpoint" validate:"required,uri"` // TODO maybe we can do a redirect
}

const PAGE_SIZE = 100

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, fmt.Errorf("%w: %w", err, models.ErrInvalidConfig)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	err := validate.Struct(config)
	if err != nil {
		return Config{}, fmt.Errorf("%w: %w", err, models.ErrInvalidConfig)
	}
	return config, nil
}
