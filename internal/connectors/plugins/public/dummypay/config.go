package dummypay

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

type Config struct {
	Directory string `json:"directory"`
}

func (c Config) validate() error {
	if c.Directory == "" {
		return fmt.Errorf("missing directory in config: %w", models.ErrInvalidConfig)
	}
	return nil
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, config.validate()
}
