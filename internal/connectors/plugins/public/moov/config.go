package moov

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
)

type Config struct {
	PublicKey   string `json:"publicKey" validate:"required"`
	SecretKey   string `json:"secretKey" validate:"required"`
	Environment string `json:"environment" validate:"required,oneof=sandbox production"`
	PageSize    int    `json:"pageSize" validate:"omitempty,min=1,max=100"`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, fmt.Errorf("%w: %w", models.ErrInvalidConfig, err)
	}

	// Set default page size if not provided
	if config.PageSize == 0 {
		config.PageSize = 25
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return Config{}, fmt.Errorf("%w: %w", models.ErrInvalidConfig, err)
	}

	return config, nil
}