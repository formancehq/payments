package wise

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/formancehq/payments/internal/models"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	APIKey           string `json:"apiKey" validate:"required"`
	WebhookPublicKey string `json:"webhookPublicKey" validate:"required"`

	webhookPublicKey *rsa.PublicKey `json:"-"`
}

func (c *Config) validate() error {
	p, _ := pem.Decode([]byte(c.WebhookPublicKey))
	if p == nil {
		return fmt.Errorf("invalid webhook public key in config: %w", models.ErrInvalidConfig)
	}

	publicKey, err := x509.ParsePKIXPublicKey(p.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse webhook public key in config %w: %w", err, models.ErrInvalidConfig)
	}

	switch pub := publicKey.(type) {
	case *rsa.PublicKey:
		c.webhookPublicKey = pub
	default:
		return fmt.Errorf("invalid webhook public key in config: %w", models.ErrInvalidConfig)
	}

	return nil
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var config Config
	if err := json.Unmarshal(payload, &config); err != nil {
		return Config{}, errors.Wrap(models.ErrInvalidConfig, err.Error())
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return config, err
	}
	return config, config.validate()
}
