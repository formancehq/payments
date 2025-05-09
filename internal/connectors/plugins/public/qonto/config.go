package qonto

import (
	"encoding/json"
	"fmt"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
)

type Config struct {
	ClientID     string `json:"clientID" validate:"required"`
	APIKey       string `json:"apiKey" validate:"required"`
	Endpoint     string `json:"endpoint" validate:"required"`
	StagingToken string `json:"stagingToken" validate:"omitempty"`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, fmt.Errorf("%w: %w", err, models.ErrInvalidConfig)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
