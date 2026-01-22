package generic

import (
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

// Generic connector allows lower polling periods for dev/testing.
// The actual minimum is enforced by the --connector-polling-period-minimum flag.
const genericMinimumPollingPeriod = 1 * time.Second

type Config struct {
	APIKey        string                     `json:"apiKey" validate:"required"`
	Endpoint      string                     `json:"endpoint" validate:"required"`
	PollingPeriod sharedconfig.PollingPeriod `json:"pollingPeriod"`
}

const PAGE_SIZE = 100 // 100 seems more likely

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey        string `json:"apiKey"`
		Endpoint      string `json:"endpoint"`
		PollingPeriod string `json:"pollingPeriod"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	pp, err := sharedconfig.NewPollingPeriod(
		raw.PollingPeriod,
		sharedconfig.DefaultPollingPeriod,
		genericMinimumPollingPeriod, // Allow low polling for generic connector; real minimum enforced by manager
	)
	if err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	config := Config{
		APIKey:        raw.APIKey,
		Endpoint:      raw.Endpoint,
		PollingPeriod: pp,
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	return config, validate.Struct(config)
}
