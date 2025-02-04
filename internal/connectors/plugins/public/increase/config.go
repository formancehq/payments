package increase

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/formancehq/payments/internal/models"
)

type Config struct {
	APIKey      string `json:"apiKey" validate:"required"`
	Environment string `json:"environment" validate:"required,oneof=sandbox production"`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, fmt.Errorf("%w: %w", err, models.ErrInvalidConfig)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
