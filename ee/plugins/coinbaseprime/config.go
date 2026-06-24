package coinbaseprime

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	APIKey      string `json:"apiKey" validate:"required"`
	APISecret   string `json:"apiSecret" validate:"required"`
	Passphrase  string `json:"passphrase" validate:"required"`
	PortfolioID string `json:"portfolioId" validate:"required"`
}

const PAGE_SIZE = 100

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey      string `json:"apiKey"`
		APISecret   string `json:"apiSecret"`
		Passphrase  string `json:"passphrase"`
		PortfolioID string `json:"portfolioId"`
	}

	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	config := Config{
		APIKey:      raw.APIKey,
		APISecret:   raw.APISecret,
		Passphrase:  raw.Passphrase,
		PortfolioID: raw.PortfolioID,
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	return config, nil
}
