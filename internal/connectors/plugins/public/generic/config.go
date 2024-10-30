package generic

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
)

type Config struct {
	APIKey   string `json:"apiKey"`
	Endpoint string `json:"endpoint"`
}

func (c Config) validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("missing api key in config: %w", models.ErrInvalidConfig)
	}

	if c.Endpoint == "" {
		return fmt.Errorf("missing endpoint in config: %w", models.ErrInvalidConfig)
	}

	return nil
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	return config, config.validate()
}
