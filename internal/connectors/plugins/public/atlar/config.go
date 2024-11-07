package atlar

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
	"github.com/pkg/errors"
)

type Config struct {
	BaseURL   string `json:"baseURL"`
	AccessKey string `json:"accessKey"`
	Secret    string `json:"secret"`
}

func (c Config) validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("missing baseURL in config: %w", models.ErrInvalidConfig)
	}

	if c.AccessKey == "" {
		return fmt.Errorf("missing access key in config: %w", models.ErrInvalidConfig)
	}

	if c.Secret == "" {
		return fmt.Errorf("missing secret in config: %w", models.ErrInvalidConfig)
	}

	return nil
}

func unmarshalAndValidateConfig(payload []byte) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	return config, config.validate()
}
