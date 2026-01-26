package kraken

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	Endpoint      string                     `json:"endpoint" validate:"required"`
	PublicKey     string                     `json:"publicKey" validate:"required"`
	PrivateKey    string                     `json:"privateKey" validate:"required"`
	PollingPeriod sharedconfig.PollingPeriod `json:"pollingPeriod"`
}

const PAGE_SIZE = 50

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		Endpoint      string `json:"endpoint"`
		PublicKey     string `json:"publicKey"`
		PrivateKey    string `json:"privateKey"`
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
		Endpoint:      raw.Endpoint,
		PublicKey:     raw.PublicKey,
		PrivateKey:    raw.PrivateKey,
		PollingPeriod: pp,
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	return config, nil
}
