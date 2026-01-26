package routable

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
)

type Config struct {
	APIToken            string `json:"apiToken" validate:"required"`
	Endpoint            string `json:"endpoint" validate:"required,url"`
	WebhookSharedSecret string `json:"webhookSharedSecret" validate:"omitempty"`
	ActingTeamMemberID  string `json:"actingTeamMemberID" validate:"omitempty"`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, fmt.Errorf("%w: %w", err, models.ErrInvalidConfig)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
