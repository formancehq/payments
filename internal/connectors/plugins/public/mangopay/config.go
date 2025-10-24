package mangopay

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

 type Config struct {
	ClientID      string                     `json:"clientID" validate:"required"`
	APIKey        string                     `json:"apiKey" validate:"required"`
	Endpoint      string                     `json:"endpoint" validate:"required"`
	PollingPeriod sharedconfig.PollingPeriod `json:"pollingPeriod"`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		ClientID      string `json:"clientID"`
		APIKey        string `json:"apiKey"`
		Endpoint      string `json:"endpoint"`
		PollingPeriod string `json:"pollingPeriod"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	pp, err := sharedconfig.NewPollingPeriod(
		raw.PollingPeriod,
		sharedconfig.DefaultPollingPeriod,
		sharedconfig.MinimumPollingPeriod,
	)
	if err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	config := Config{
		ClientID:      raw.ClientID,
		APIKey:        raw.APIKey,
		Endpoint:      raw.Endpoint,
		PollingPeriod: pp,
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
