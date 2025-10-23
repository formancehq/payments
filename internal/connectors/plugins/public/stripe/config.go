package stripe

import (
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

const (
	minimumPollingInterval = 20 * time.Minute
	defaultPollingInterval = 30 * time.Minute
)

type Config struct {
	APIKey        string        `json:"apiKey" validate:"required"`
	PollingPeriod time.Duration `json:"pollingPeriod" validate:"required,gte=0s"`
}

func (c Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		APIKey        string `json:"apiKey"`
		PollingPeriod string `json:"pollingPeriod"`
	}{
		APIKey:        c.APIKey,
		PollingPeriod: c.PollingPeriod.String(),
	})
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey        string `json:"apiKey"`
		PollingPeriod string `json:"pollingPeriod"`
	}

	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	pollingPeriod := defaultPollingInterval
	if raw.PollingPeriod != "" {
		var err error
		pollingPeriod, err = time.ParseDuration(raw.PollingPeriod)
		if err != nil {
			return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
		}
	}

	validate := validator.New(validator.WithRequiredStructEnabled())

	if pollingPeriod < minimumPollingInterval {
		pollingPeriod = minimumPollingInterval
	}

	config := Config{
		APIKey:        raw.APIKey,
		PollingPeriod: pollingPeriod,
	}

	return config, validate.Struct(config)
}
