package checkout

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/formancehq/payments/internal/models"
)

type Config struct {
	// This is the config a user will pass when installing this connector.
	// Authentication criteria for connecting to your connector should be provided here. Example:
	Environment string `json:"environment" validate:"required"`
	OAuthClientID   string `json:"oauthClientID" validate:"required"`
	OAuthClientSecret   string `json:"oauthClientSecret" validate:"required"`
	EntityID   string `json:"entityId" validate:"required"`
	ProcessingChannelId string `json:"processingChannelId" validate:"required"`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, fmt.Errorf("%w: %w", err, models.ErrInvalidConfig)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
