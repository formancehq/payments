package atlar

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/connector"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	BaseURL       string                     `json:"baseUrl" validate:"required"`
	AccessKey     string                     `json:"accessKey" validate:"required"`
	Secret        string                     `json:"secret" validate:"required"`
	PollingPeriod connector.PollingPeriod `json:"pollingPeriod"`
}

const PAGE_SIZE = 100 // max size is 500 according to docs

func unmarshalAndValidateConfig(payload []byte) (Config, error) {
	var raw struct {
		BaseURL       string `json:"baseUrl"`
		AccessKey     string `json:"accessKey"`
		Secret        string `json:"secret"`
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
		BaseURL:       raw.BaseURL,
		AccessKey:     raw.AccessKey,
		Secret:        raw.Secret,
		PollingPeriod: pp,
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
