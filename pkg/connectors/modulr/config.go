package modulr

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/connector"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	APIKey        string                     `json:"apiKey" validate:"required"`
	APISecret     string                     `json:"apiSecret" validate:"required"`
	Endpoint      string                     `json:"endpoint" validate:"required"`
	PollingPeriod connector.PollingPeriod `json:"pollingPeriod"`
}

const PAGE_SIZE = 100 // max page size is 500

func unmarshalAndValidateConfig(payload []byte) (Config, error) {
	var raw struct {
		APIKey        string `json:"apiKey"`
		APISecret     string `json:"apiSecret"`
		Endpoint      string `json:"endpoint"`
		PollingPeriod string `json:"pollingPeriod"`
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
		APIKey:        raw.APIKey,
		APISecret:     raw.APISecret,
		Endpoint:      raw.Endpoint,
		PollingPeriod: pp,
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
