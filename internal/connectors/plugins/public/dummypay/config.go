package dummypay

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	Directory           string `json:"directory" validate:"required,dirpath"`
	CreateLinkFlowError bool   `json:"linkFlowError" validate:""`
	UpdateLinkFlowError bool   `json:"updateLinkFlowError" validate:""`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
