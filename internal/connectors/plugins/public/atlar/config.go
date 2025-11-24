package atlar

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	BaseURL       string                     `json:"baseUrl" validate:"required"`
	AccessKey     string                     `json:"accessKey" validate:"required"`
	Secret        string                     `json:"secret" validate:"required"`
	PollingPeriod sharedconfig.PollingPeriod `json:"pollingPeriod"`
}

const PAGE_SIZE = 25

func unmarshalAndValidateConfig(payload []byte) (Config, error) {
	var raw struct {
		BaseURL       string `json:"baseUrl"`
		AccessKey     string `json:"accessKey"`
		Secret        string `json:"secret"`
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
		BaseURL:       raw.BaseURL,
		AccessKey:     raw.AccessKey,
		Secret:        raw.Secret,
		PollingPeriod: pp,
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
