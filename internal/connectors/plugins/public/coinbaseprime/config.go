package coinbaseprime

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
)

type Config struct {
	// TODO: fill Config struct
	// This is the config a user will pass when installing this connector.
	// Authentication criteria for connecting to your connector should be provided here. Example:
	// ClientID string `json:"clientID" validate:"required"`
	// APIKey   string `json:"apiKey" validate:"required"`
	// Endpoint string `json:"endpoint" validate:"required"`
	Credentials string `json:"credentials" validate:"required"`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, fmt.Errorf("%w: %w", err, models.ErrInvalidConfig)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
