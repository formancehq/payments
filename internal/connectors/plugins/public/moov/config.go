package moov

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
)

type Config struct {
	PublicKey  string `json:"publicKey" validate:"required"`
	PrivateKey string `json:"privateKey" validate:"required"`
	Endpoint   string `json:"endpoint" validate:"required,url"`
	AccountID  string `json:"accountID" validate:"required"`
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, fmt.Errorf("%w: %w", err, models.ErrInvalidConfig)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return Config{}, err
	}

	endpoint, err := url.Parse(config.Endpoint)
	if err != nil {
		return Config{}, fmt.Errorf("%w: %w", err, models.ErrInvalidConfig)
	}
	config.Endpoint = endpoint.Host + endpoint.Path

	return config, nil
}
