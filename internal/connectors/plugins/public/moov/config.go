package moov

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
)

type Config struct {
	PublicKey  string `json:"publicKey" validate:"required"`
	PrivateKey string `json:"privateKey" validate:"required"`
	Endpoint   string `json:"endpoint" validate:"required"`
	AccountID  string `json:"accountID" validate:"required"`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, fmt.Errorf("%w: %w", err, models.ErrInvalidConfig)
	}

	endpoint := strings.TrimPrefix(config.Endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	config.Endpoint = endpoint

	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
