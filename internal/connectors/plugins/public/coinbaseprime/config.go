package coinbaseprime

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	APIKey        string                     `json:"apiKey" validate:"required"`
	APISecret     string                     `json:"apiSecret" validate:"required"`
	Passphrase    string                     `json:"passphrase" validate:"required"`
	PortfolioID   string                     `json:"portfolioId" validate:"required"`
	PollingPeriod sharedconfig.PollingPeriod `json:"pollingPeriod"`
}

const PAGE_SIZE = 100

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey        string `json:"apiKey"`
		APISecret     string `json:"apiSecret"`
		Passphrase    string `json:"passphrase"`
		PortfolioID   string `json:"portfolioId"`
		PollingPeriod string `json:"pollingPeriod"`
	}

	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	pp, err := sharedconfig.NewPollingPeriod(
		raw.PollingPeriod,
		sharedconfig.DefaultPollingPeriod,
		sharedconfig.MinimumPollingPeriod,
	)
	if err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	config := Config{
		APIKey:        raw.APIKey,
		APISecret:     raw.APISecret,
		Passphrase:    raw.Passphrase,
		PortfolioID:   raw.PortfolioID,
		PollingPeriod: pp,
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	return config, nil
}
