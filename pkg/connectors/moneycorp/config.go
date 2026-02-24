package moneycorp

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/connector"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	ClientID      string                     `json:"clientID" validate:"required"`
	APIKey        string                     `json:"apiKey" validate:"required"`
	Endpoint      string                     `json:"endpoint" validate:"required"`
	PollingPeriod connector.PollingPeriod `json:"pollingPeriod"`
}

const PAGE_SIZE = 100 // max page size is 10000 according to docs (!)

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		ClientID      string `json:"clientID"`
		APIKey        string `json:"apiKey"`
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
		ClientID:      raw.ClientID,
		APIKey:        raw.APIKey,
		Endpoint:      raw.Endpoint,
		PollingPeriod: pp,
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
