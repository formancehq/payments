package coinbaseprime

import (
	"encoding/json"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	AccessKey     string                     `json:"accessKey" validate:"required"`
	Passphrase    string                     `json:"passphrase" validate:"required"`
	SigningKey    string                     `json:"signingKey" validate:"required"`
	PortfolioID   string                     `json:"portfolioId" validate:"required"`
	SvcAccountID  string                     `json:"svcAccountId" validate:"required"`
	EntityID      string                     `json:"entityId" validate:"required"`
	PollingPeriod sharedconfig.PollingPeriod `json:"pollingPeriod"`
}

const PAGE_SIZE = 100

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		AccessKey     string `json:"accessKey"`
		Passphrase    string `json:"passphrase"`
		SigningKey    string `json:"signingKey"`
		PortfolioID   string `json:"portfolioId"`
		SvcAccountID  string `json:"svcAccountId"`
		EntityID      string `json:"entityId"`
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
		AccessKey:     raw.AccessKey,
		Passphrase:    raw.Passphrase,
		SigningKey:    raw.SigningKey,
		PortfolioID:   raw.PortfolioID,
		SvcAccountID:  raw.SvcAccountID,
		EntityID:      raw.EntityID,
		PollingPeriod: pp,
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	return config, nil
}
