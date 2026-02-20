package fireblocks

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/formancehq/payments/pkg/connector"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Config struct {
	APIKey        string                  `json:"apiKey" validate:"required"`
	PrivateKey    string                  `json:"privateKey" validate:"required"`
	Endpoint      string                  `json:"endpoint"`
	PollingPeriod connector.PollingPeriod `json:"pollingPeriod"`

	privateKey *rsa.PrivateKey `json:"-"`
}

const (
	PAGE_SIZE      = 200
	DefaultEndpoint = "https://api.fireblocks.io"
)

func (c *Config) validate() error {
	p, _ := pem.Decode([]byte(c.PrivateKey))
	if p == nil {
		return connector.NewWrappedError(
			fmt.Errorf("invalid private key PEM in config"),
			connector.ErrInvalidConfig,
		)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(p.Bytes)
	if err != nil {
		// Try PKCS8 format
		key, err2 := x509.ParsePKCS8PrivateKey(p.Bytes)
		if err2 != nil {
			return connector.NewWrappedError(
				fmt.Errorf("failed to parse private key as PKCS1 (%w) or PKCS8 (%v)", err, err2),
				connector.ErrInvalidConfig,
			)
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return connector.NewWrappedError(
				fmt.Errorf("private key is not RSA"),
				connector.ErrInvalidConfig,
			)
		}
		privateKey = rsaKey
	}

	c.privateKey = privateKey

	if c.Endpoint == "" {
		c.Endpoint = DefaultEndpoint
	}

	return nil
}

func unmarshalAndValidateConfig(payload json.RawMessage) (Config, error) {
	var raw struct {
		APIKey        string `json:"apiKey"`
		PrivateKey    string `json:"privateKey"`
		Endpoint      string `json:"endpoint"`
		PollingPeriod string `json:"pollingPeriod"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Config{}, errors.Wrap(connector.ErrInvalidConfig, err.Error())
	}

	pp, err := connector.NewPollingPeriod(
		raw.PollingPeriod,
		connector.DefaultPollingPeriod,
		connector.MinimumPollingPeriod,
	)
	if err != nil {
		return Config{}, errors.Wrap(connector.ErrInvalidConfig, err.Error())
	}

	config := Config{
		APIKey:        raw.APIKey,
		PrivateKey:    raw.PrivateKey,
		Endpoint:      raw.Endpoint,
		PollingPeriod: pp,
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(config); err != nil {
		return config, errors.Wrap(connector.ErrInvalidConfig, err.Error())
	}

	return config, config.validate()
}
