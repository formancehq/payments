package currencycloud

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/connector"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	LoginID  string `json:"loginID" validate:"required"`
	APIKey   string `json:"apiKey" validate:"required"`
	Endpoint string `json:"endpoint" validate:"required"`
}

const PAGE_SIZE = 25 // Limit is undocumented.

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, errors.Wrap(connector.ErrInvalidConfig, err.Error())
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
