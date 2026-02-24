package increase

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/connector"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	APIKey              string                     `json:"apiKey" validate:"required"`
	Endpoint            string                     `json:"endpoint" validate:"required"`
	WebhookSharedSecret string                     `json:"webhookSharedSecret" validate:"required"`
	PollingPeriod       connector.PollingPeriod `json:"pollingPeriod"`
}

const PAGE_SIZE = 100 // max size is 100

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey              string `json:"apiKey"`
		Endpoint            string `json:"endpoint"`
		WebhookSharedSecret string `json:"webhookSharedSecret"`
		PollingPeriod       string `json:"pollingPeriod"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(connector.ErrInvalidConfig, err.Error())
	}

	pp, err := connector.NewPollingPeriod(
		raw.PollingPeriod,
		connector.DefaultPollingPeriod,
		connector.MinimumPollingPeriod,
	)
	if err != nil {
		return Config{}, errors.Wrap(connector.ErrInvalidConfig, err.Error())
	}

	config := Config{
		APIKey:              raw.APIKey,
		Endpoint:            raw.Endpoint,
		WebhookSharedSecret: raw.WebhookSharedSecret,
		PollingPeriod:       pp,
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
