package krakenpro

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	APIKey    string `json:"apiKey" validate:"required"`
	APISecret string `json:"apiSecret" validate:"required"`
	Endpoint  string `json:"endpoint" validate:"required,url"`
}

// PAGE_SIZE is the per-call page bound for Ledgers / ClosedOrders.
// Kraken documents no hard cap; 50 is the observed default and also the
// short-page signal the frozen-window walk uses to detect drain.
const PAGE_SIZE = 50

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	return config, nil
}
