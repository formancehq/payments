package bitstamp

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
)

// Config defines the connector configuration for Bitstamp.
// - Endpoint: Bitstamp API base URL (usually https://www.bitstamp.net)
// - Accounts: List of sub-accounts with their API credentials
type Config struct {
	Endpoint string           `json:"endpoint" validate:"required"`
	Accounts []client.Account `json:"accounts" validate:"required,dive"`
}

// unmarshalAndValidateConfig parses and validates the connector configuration.
func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, fmt.Errorf("%w: %w", err, models.ErrInvalidConfig)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
