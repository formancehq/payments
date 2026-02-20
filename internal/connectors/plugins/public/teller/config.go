package teller

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

const PAGE_SIZE = 100

type Config struct {
	ApplicationID string `json:"applicationID" validate:"required"`
	IsSandbox     bool   `json:"isSandbox" validate:""`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
