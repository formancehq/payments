package adyen

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	APIKey             string `json:"apiKey" validate:"required"`
	CompanyID          string `json:"companyID" validate:"required"`
	LiveEndpointPrefix string `json:"liveEndpointPrefix" validate:"omitempty,url_encoded"`

	// https://datatracker.ietf.org/doc/html/rfc7617
	WebhookUsername string `json:"webhookUsername" validate:"omitempty,excludes=:"`
	WebhookPassword string `json:"webhookPassword" validate:""`
}

func unmarshalAndValidateConfig(payload []byte) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
