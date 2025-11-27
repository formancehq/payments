package wise

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	APIKey           string                     `json:"apiKey" validate:"required"`
	WebhookPublicKey string                     `json:"webhookPublicKey" validate:"required"`
	PollingPeriod    sharedconfig.PollingPeriod `json:"pollingPeriod"`

	webhookPublicKey *rsa.PublicKey `json:"-"`
}

const PAGE_SIZE = 100 // max page size is 100

func (c *Config) validate() error {
	p, _ := pem.Decode([]byte(c.WebhookPublicKey))
	if p == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("invalid webhook public key in config"),
			models.ErrInvalidConfig,
		)
	}

	publicKey, err := x509.ParsePKIXPublicKey(p.Bytes)
	if err != nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("failed to parse webhook public key in config %w", err),
			models.ErrInvalidConfig,
		)
	}

	switch pub := publicKey.(type) {
	case *rsa.PublicKey:
		c.webhookPublicKey = pub
	default:
		return errorsutils.NewWrappedError(
			fmt.Errorf("invalid webhook public key in config"),
			models.ErrInvalidConfig,
		)
	}

	return nil
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey           string `json:"apiKey"`
		WebhookPublicKey string `json:"webhookPublicKey"`
		PollingPeriod    string `json:"pollingPeriod"`
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
		APIKey:           raw.APIKey,
		WebhookPublicKey: raw.WebhookPublicKey,
		PollingPeriod:    pp,
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return config, err
	}
	return config, config.validate()
}
