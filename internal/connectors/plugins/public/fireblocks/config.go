package fireblocks

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
	APIKey        string                     `json:"apiKey" validate:"required"`
	PrivateKey    string                     `json:"privateKey" validate:"required"`
	BaseURL       string                     `json:"baseURL"`
	PollingPeriod sharedconfig.PollingPeriod `json:"pollingPeriod"`

	privateKey *rsa.PrivateKey `json:"-"`
}

const (
	PAGE_SIZE      = 200
	DefaultBaseURL = "https://api.fireblocks.io"
)

func (c *Config) validate() error {
	p, _ := pem.Decode([]byte(c.PrivateKey))
	if p == nil {
		return errorsutils.NewWrappedError(
			fmt.Errorf("invalid private key PEM in config"),
			models.ErrInvalidConfig,
		)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(p.Bytes)
	if err != nil {
		// Try PKCS8 format
		key, err2 := x509.ParsePKCS8PrivateKey(p.Bytes)
		if err2 != nil {
			return errorsutils.NewWrappedError(
				fmt.Errorf("failed to parse private key: %w", err),
				models.ErrInvalidConfig,
			)
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return errorsutils.NewWrappedError(
				fmt.Errorf("private key is not RSA"),
				models.ErrInvalidConfig,
			)
		}
		privateKey = rsaKey
	}

	c.privateKey = privateKey

	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}

	return nil
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey        string `json:"apiKey"`
		PrivateKey    string `json:"privateKey"`
		BaseURL       string `json:"baseURL"`
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
		PrivateKey:    raw.PrivateKey,
		BaseURL:       raw.BaseURL,
		PollingPeriod: pp,
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return config, err
	}

	return config, config.validate()
}
