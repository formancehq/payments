package atlar

import (
	"encoding/json"

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
		return errors.Wrap(models.ErrInvalidConfig, "missing baseURL in config")
	}

	if c.AccessKey == "" {
		return errors.Wrap(models.ErrInvalidConfig, "missing access key in config")
	}

	if c.Secret == "" {
		return errors.Wrap(models.ErrInvalidConfig, "missing secret in config")
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
