package fireblocks

import (
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	APIKey        string                     `json:"apiKey" validate:"required"`
	PrivateKey    string                     `json:"privateKey" validate:"required"`
	BaseURL       string                     `json:"baseUrl"`
	Sandbox       bool                       `json:"sandbox"`
	PollingPeriod sharedconfig.PollingPeriod `json:"pollingPeriod"`
}

const (
	defaultBaseURL = "https://api.fireblocks.io/v1"
	sandboxBaseURL = "https://sandbox-api.fireblocks.io/v1"
	PAGE_SIZE      = 100
)

var (
	defaultPollingPeriod = 2 * time.Minute
	minimumPollingPeriod = 30 * time.Second
)

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey        string `json:"apiKey"`
		PrivateKey    string `json:"privateKey"`
		BaseURL       string `json:"baseUrl"`
		Sandbox       bool   `json:"sandbox"`
		PollingPeriod string `json:"pollingPeriod"`
	}

	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	pp, err := sharedconfig.NewPollingPeriod(
		raw.PollingPeriod,
		defaultPollingPeriod,
		minimumPollingPeriod,
	)
	if err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	baseURL := raw.BaseURL
	if baseURL == "" {
		if raw.Sandbox {
			baseURL = sandboxBaseURL
		} else {
			baseURL = defaultBaseURL
		}
	}

	config := Config{
		APIKey:        raw.APIKey,
		PrivateKey:    raw.PrivateKey,
		BaseURL:       baseURL,
		Sandbox:       raw.Sandbox,
		PollingPeriod: pp,
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}

	return config, nil
}
